package api

import (
	"io/fs"
	"net/http"
	"os"

	"gatus/v5/config"
	"gatus/v5/config/ui"
	"gatus/v5/config/web"
	"gatus/v5/liveupdates"
	static "gatus/v5/web"
	"github.com/TwiN/health"
	"github.com/TwiN/logr"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberfs "github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/redirect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	router *fiber.App
}

func New(cfg *config.Config) *API {
	api := &API{}
	if cfg.Web == nil {
		logr.Warnf("[api.New] nil web config passed as parameter. This should only happen in tests. Using default web configuration")
		cfg.Web = web.GetDefaultConfig()
	}
	if cfg.UI == nil {
		logr.Warnf("[api.New] nil ui config passed as parameter. This should only happen in tests. Using default ui configuration")
		cfg.UI = ui.GetDefaultConfig()
	}
	api.router = api.createRouter(cfg)
	return api
}

func (a *API) Router() *fiber.App {
	return a.router
}

func (a *API) createRouter(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logr.Errorf("[api.ErrorHandler] %s", err.Error())
			return fiber.DefaultErrorHandler(c, err)
		},
		ReadBufferSize: cfg.Web.ReadBufferSize,
		Network:        fiber.NetworkTCP,
		Immutable:      true, // If not enabled, will cause issues due to fiber's zero allocation. See #1268 and https://docs.gofiber.io/#zero-allocation
	})
	if os.Getenv("ENVIRONMENT") == "dev" {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     "http://localhost:8081",
			AllowCredentials: true,
		}))
	}
	// Middlewares
	app.Use(recover.New())
	// Fork: the event streams are never compressed, which would hold their events in a buffer
	app.Use(compress.New(compress.Config{Next: func(c *fiber.Ctx) bool {
		return liveupdates.IsEventsPath(c.Path())
	}}))
	// Define metrics handler, if necessary
	if cfg.Metrics {
		metricsHandler := promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			DisableCompression: true,
		}))
		app.Get("/metrics", adaptor.HTTPHandler(metricsHandler))
	}
	// Define main router
	apiRouter := app.Group("/api")
	////////////////////////
	// UNPROTECTED ROUTES //
	////////////////////////
	unprotectedAPIRouter := apiRouter.Group("/")
	// IP address of the client for the failure limiter of security.basic (fork), see api/auth.go
	clientIP := clientIPMiddleware(cfg.StatusPages.TrustedProxyPrefixes())
	unprotectedAPIRouter.Get("/v1/config", clientIP, ConfigHandler{securityConfig: cfg.Security, config: cfg}.GetConfig)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/health/badge.svg", HealthBadge)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/health/badge.shields", HealthBadgeShields)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/uptimes/:duration", UptimeRaw)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/uptimes/:duration/badge.svg", UptimeBadge)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/response-times/:duration", ResponseTimeRaw)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/response-times/:duration/badge.svg", ResponseTimeBadge(cfg))
	unprotectedAPIRouter.Get("/v1/endpoints/:key/response-times/:duration/chart.svg", ResponseTimeChart)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/response-times/:duration/history", ResponseTimeHistory)
	// This endpoint requires authz with bearer token, so technically it is protected
	unprotectedAPIRouter.Post("/v1/endpoints/:key/external", CreateExternalEndpointResult(cfg))
	// Push compatible with the Uptime Kuma (fork): routes and catch-alls, see api/push.go
	registerPushRoutes(unprotectedAPIRouter, cfg)
	// Public status pages (fork): API, catch-all and SPA routes, see api/status_page.go
	registerStatusPageRoutes(app, unprotectedAPIRouter, cfg)
	// Login screen of security.basic (fork): login and logout, see api/auth.go
	registerAuthRoutes(unprotectedAPIRouter, cfg, clientIP)
	// SPA
	app.Get("/", SinglePageApplication(cfg.UI))
	if cfg.Security.UsesBasicLogin() {
		app.Get("/login", SinglePageApplication(cfg.UI))
	}
	app.Get("/endpoints/:key", SinglePageApplication(cfg.UI))
	app.Get("/suites/:key", SinglePageApplication(cfg.UI))
	if cfg.Admin.IsEnabled() {
		app.Get("/admin", SinglePageApplication(cfg.UI))
		app.Get("/admin/endpoints/new", SinglePageApplication(cfg.UI))
		app.Get("/admin/endpoints/:endpointKey/edit", SinglePageApplication(cfg.UI))
		app.Get("/admin/status-pages", SinglePageApplication(cfg.UI))
		app.Get("/admin/status-pages/new", SinglePageApplication(cfg.UI))
		app.Get("/admin/status-pages/:slug/edit", SinglePageApplication(cfg.UI))
		app.Get("/admin/push-keys", SinglePageApplication(cfg.UI))
	}
	// Health endpoint
	healthHandler := health.Handler().WithJSON(true)
	app.Get("/health", func(c *fiber.Ctx) error {
		statusCode, body := healthHandler.GetResponseStatusCodeAndBody()
		return c.Status(statusCode).Send(body)
	})
	// Custom CSS
	app.Get("/css/custom.css", CustomCSSHandler{customCSS: cfg.UI.CustomCSS}.GetCustomCSS)
	// Everything else falls back on static content
	app.Use(redirect.New(redirect.Config{
		Rules: map[string]string{
			"/index.html": "/",
		},
		StatusCode: 301,
	}))
	staticFileSystem, err := fs.Sub(static.FileSystem, static.RootPath)
	if err != nil {
		panic(err)
	}
	app.Use("/", fiberfs.New(fiberfs.Config{
		Root:   http.FS(staticFileSystem),
		Index:  "index.html",
		Browse: true,
	}))
	//////////////////////
	// PROTECTED ROUTES //
	//////////////////////
	// ORDER IS IMPORTANT: all routes applied AFTER the security middleware will require authn
	protectedAPIRouter := apiRouter.Group("/")
	if cfg.Security != nil {
		if err := cfg.Security.RegisterHandlers(app); err != nil {
			panic(err)
		}
		if cfg.Security.UsesBasicLogin() {
			protectedAPIRouter.Use(clientIP)
		}
		if err := cfg.Security.ApplySecurityMiddleware(protectedAPIRouter); err != nil {
			panic(err)
		}
	}
	protectedAPIRouter.Get("/v1/endpoints/statuses", EndpointStatuses(cfg))
	protectedAPIRouter.Get("/v1/endpoints/:key/statuses", EndpointStatus(cfg))
	// Fork: notifications of the new results of an endpoint in real time, see api/live_updates.go
	protectedAPIRouter.Get("/v1/endpoints/:key/events", endpointEventsHandler(cfg))
	// Fork: data of the response time chart of the endpoint details, see api/response_time_chart.go
	protectedAPIRouter.Get("/v1/endpoints/:key/response-time-chart", endpointResponseTimeChartHandler(cfg))
	protectedAPIRouter.Get("/v1/suites/statuses", SuiteStatuses(cfg))
	protectedAPIRouter.Get("/v1/suites/:key/statuses", SuiteStatus(cfg))
	// Administration of endpoints (fork): only registered when enabled, see api/admin.go
	if cfg.Admin.IsEnabled() {
		registerAdminRoutes(protectedAPIRouter.Group("/v1/admin", cfg.Security.AdminMiddleware(cfg.Admin), adminRequestProtection(cfg.Admin)), cfg)
	}
	return app
}
