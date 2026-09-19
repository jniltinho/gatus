package api

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"

	"gatus/v5/config"
	"gatus/v5/config/ui"
	"gatus/v5/config/web"
	"gatus/v5/internal/httpx"
	"gatus/v5/liveupdates"
	static "gatus/v5/web"

	"github.com/TwiN/health"
	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	router *echo.Echo
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

// Router returns the HTTP handler of the API: an *echo.Echo, which is a net/http handler
func (a *API) Router() *echo.Echo {
	return a.router
}

func (a *API) createRouter(cfg *config.Config) *echo.Echo {
	app := echo.NewWithConfig(echo.Config{
		HTTPErrorHandler: httpErrorHandler,
		// The router is the default one on purpose. It matches the path as it was sent (URL.RawPath when there is one) and
		// does not unescape the parameters, so every handler unescapes its key once, as it did with Fiber, and a key with
		// %2F or %252F means what it always meant. RouterConfig.UseEscapedPathForMatching does the opposite of what its
		// name suggests in v5.3: turning it on matches the DECODED path, and core%2Fapi becomes two segments.
		// IPExtractor is never set: the IP address of a client only comes from the connection, see httpx.RemoteIP.
		// RouterConfig.AutoHandleHEAD stays off: it would open an event stream for a HEAD, see httpx.GetAndHead.
	})
	// Fiber ignored a trailing slash. It has to be a Pre middleware: once the router has picked a route it is too late.
	app.Pre(middleware.RemoveTrailingSlash())
	if os.Getenv("ENVIRONMENT") == "dev" {
		app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"http://localhost:8081"},
			AllowCredentials: true,
			// Fork: the name of a downloaded backup
			ExposeHeaders: []string{"Content-Disposition"},
		}))
	}
	// Middlewares
	app.Use(middleware.Recover())
	// The body is read ahead and refused above 4 MiB before any handler runs, see httpx.BufferBody
	app.Use(httpx.BufferBody)
	// Fork: the event streams are never compressed, which would hold their events in a buffer. The skipper looks at the
	// path of the request: echo.Context.Path is the registered route.
	app.Use(middleware.GzipWithConfig(middleware.GzipConfig{Skipper: func(c *echo.Context) bool {
		return liveupdates.IsEventsPath(httpx.Path(c))
	}}))
	// Define metrics handler, if necessary
	if cfg.Metrics {
		metricsHandler := promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			DisableCompression: true,
		}))
		httpx.GetAndHead(app, "/metrics", echo.WrapHandler(metricsHandler))
	}
	////////////////////////
	// UNPROTECTED ROUTES //
	////////////////////////
	// Two explicit groups under /api: with Echo the protection is a property of the group, not of the order in which the
	// routes are registered
	unprotectedAPIRouter := app.Group("/api")
	// IP address of the client for the failure limiter of security.basic (fork), see api/auth.go
	clientIP := clientIPMiddleware(cfg.StatusPages.TrustedProxyPrefixes())
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/config", ConfigHandler{securityConfig: cfg.Security, config: cfg}.GetConfig, clientIP)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/health/badge.svg", HealthBadge)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/health/badge.shields", HealthBadgeShields)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/uptimes/:duration", UptimeRaw)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/uptimes/:duration/badge.svg", UptimeBadge)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/response-times/:duration", ResponseTimeRaw)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/response-times/:duration/badge.svg", ResponseTimeBadge(cfg))
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/response-times/:duration/chart.svg", ResponseTimeChart)
	httpx.GetAndHead(unprotectedAPIRouter, "/v1/endpoints/:key/response-times/:duration/history", ResponseTimeHistory)
	// This endpoint requires authz with bearer token, so technically it is protected
	unprotectedAPIRouter.POST("/v1/endpoints/:key/external", CreateExternalEndpointResult(cfg))
	// Push compatible with the Uptime Kuma (fork): routes and catch-alls, see api/push.go
	registerPushRoutes(unprotectedAPIRouter, cfg)
	// Public status pages (fork): API, catch-all and SPA routes, see api/status_page.go
	registerStatusPageRoutes(app, unprotectedAPIRouter, cfg)
	// Login screen of security.basic (fork): login and logout, see api/auth.go
	registerAuthRoutes(unprotectedAPIRouter, cfg, clientIP)
	// SPA
	httpx.GetAndHead(app, "/", SinglePageApplication(cfg.UI))
	if cfg.Security.UsesBasicLogin() {
		httpx.GetAndHead(app, "/login", SinglePageApplication(cfg.UI))
	}
	httpx.GetAndHead(app, "/endpoints/:key", SinglePageApplication(cfg.UI))
	httpx.GetAndHead(app, "/suites/:key", SinglePageApplication(cfg.UI))
	if cfg.Admin.IsEnabled() {
		httpx.GetAndHead(app, "/admin", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/endpoints/new", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/endpoints/:endpointKey/edit", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/status-pages", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/status-pages/new", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/status-pages/:slug/edit", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/push-keys", SinglePageApplication(cfg.UI))
		httpx.GetAndHead(app, "/admin/backup", SinglePageApplication(cfg.UI))
	}
	// Health endpoint
	healthHandler := health.Handler().WithJSON(true)
	httpx.GetAndHead(app, "/health", func(c *echo.Context) error {
		statusCode, body := healthHandler.GetResponseStatusCodeAndBody()
		return httpx.Send(c, statusCode, body)
	})
	// Custom CSS
	httpx.GetAndHead(app, "/css/custom.css", CustomCSSHandler{customCSS: cfg.UI.CustomCSS}.GetCustomCSS)
	// Everything else falls back on static content
	httpx.GetAndHead(app, "/index.html", func(c *echo.Context) error {
		// With its query, as the redirect of Fiber did
		target := "/"
		if query := c.Request().URL.RawQuery; len(query) > 0 {
			target += "?" + query
		}
		return c.Redirect(http.StatusMovedPermanently, target)
	})
	staticFileSystem, err := fs.Sub(static.FileSystem, static.RootPath)
	if err != nil {
		panic(err)
	}
	// The static files are the LAST resort of the router, a route on /*, and never a middleware: echo's Static middleware
	// runs before the routes, so it would answer / and /index.html with the raw template of the SPA instead of the rendered
	// page. A wildcard route loses to every other route. Without a listing of the directories, which Fiber had (Browse)
	// and nothing used: it only listed the files of the build.
	httpx.GetAndHead(app, "/*", staticFileHandler(staticFileSystem))
	//////////////////////
	// PROTECTED ROUTES //
	//////////////////////
	// A group of its own, with the security middleware. Echo registers the 404 of a group with middlewares under the
	// group, so an unknown path under /api still answers 401 without credentials, as it did when the order of the
	// registrations decided it; the catch-alls of the status pages and of the push are more specific and stay public.
	protectedAPIRouter := app.Group("/api")
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
	httpx.GetAndHead(protectedAPIRouter, "/v1/endpoints/statuses", EndpointStatuses(cfg))
	httpx.GetAndHead(protectedAPIRouter, "/v1/endpoints/:key/statuses", EndpointStatus(cfg))
	// Fork: notifications of the new results of an endpoint in real time, see api/live_updates.go
	httpx.GetAndHead(protectedAPIRouter, "/v1/endpoints/:key/events", endpointEventsHandler(cfg))
	// Fork: data of the response time chart of the endpoint details, see api/response_time_chart.go
	httpx.GetAndHead(protectedAPIRouter, "/v1/endpoints/:key/response-time-chart", endpointResponseTimeChartHandler(cfg))
	httpx.GetAndHead(protectedAPIRouter, "/v1/suites/statuses", SuiteStatuses(cfg))
	httpx.GetAndHead(protectedAPIRouter, "/v1/suites/:key/statuses", SuiteStatus(cfg))
	// Administration of endpoints (fork): only registered when enabled, see api/admin.go
	if cfg.Admin.IsEnabled() {
		registerAdminRoutes(protectedAPIRouter.Group("/v1/admin", cfg.Security.AdminMiddleware(cfg.Admin), adminRequestProtection(cfg.Admin)), cfg)
	}
	return app
}

// staticFileHandler serves a file of the embedded file system, and answers 404 for anything else. A directory is not
// found: echo's own handler redirects it to the same path with a trailing slash, which RemoveTrailingSlash takes away
// again, and the browser would go round in circles.
func staticFileHandler(fileSystem fs.FS) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// The wildcard is the escaped path, because the router matches the path as it was sent
		name, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return echo.ErrNotFound
		}
		name = path.Clean("/" + name)[1:]
		if info, statErr := fs.Stat(fileSystem, name); len(name) == 0 || statErr != nil || info.IsDir() {
			return echo.ErrNotFound
		}
		return c.FileFS(name, fileSystem)
	}
}

// httpErrorHandler answers the errors of the router with the bodies Fiber used to answer, so that a client sees the
// same 404 and 405. It never touches an answer that has already started, an event stream above all.
func httpErrorHandler(c *echo.Context, err error) {
	if response, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil && response.Committed {
		return
	}
	status := http.StatusInternalServerError
	// The errors of the router (echo.ErrNotFound, echo.ErrMethodNotAllowed) are not *echo.HTTPError in v5: the status
	// comes from the interface both implement
	var coder echo.HTTPStatusCoder
	if errors.As(err, &coder) {
		status = coder.StatusCode()
	}
	switch status {
	case http.StatusNotFound:
		_ = httpx.SendString(c, status, "Cannot "+c.Request().Method+" "+httpx.Path(c))
	case http.StatusMethodNotAllowed:
		_ = httpx.SendString(c, status, http.StatusText(status))
	default:
		logr.Errorf("[api.ErrorHandler] %s", err.Error())
		_ = httpx.SendString(c, status, err.Error())
	}
}
