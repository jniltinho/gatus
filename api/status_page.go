package api

import (
	"errors"
	"math"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"gatus/v5/config"
	"gatus/v5/statuspage"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

const (
	statusPageNotFoundBody        = `{"error":"status page not found"}`
	statusPageUnavailableBody     = `{"error":"status page temporarily unavailable"}`
	statusPageTooManyRequestsBody = `{"error":"too many requests"}`
)

// registerStatusPageRoutes registers the public routes of the status pages. They must be registered before the static
// files and the security middleware, so that they never require authentication.
//
// The catch-all of /api/v1/status-pages is always registered, even with status-pages.enabled set to false, so that no
// path under it reaches the security middleware (which would answer 401 and open the login prompt of the browser).
func registerStatusPageRoutes(app *fiber.App, unprotectedAPIRouter fiber.Router, cfg *config.Config) {
	statuspage.ConfigureLimiter(cfg.StatusPages.GetRateLimit())
	trustedProxies := cfg.StatusPages.TrustedProxyPrefixes()
	notFound := statusPageNotFound(trustedProxies)
	// Fork: the middleware of the login of a page runs on each route of a page, never on the catch-all, and captures the
	// page for the route that answers
	auth := statusPageAuth(notFound, trustedProxies)
	if cfg.StatusPages.IsEnabled() {
		unprotectedAPIRouter.Get("/v1/status-pages/:slug", auth, statusPageHandler(notFound))
		unprotectedAPIRouter.Get("/v1/status-pages/:slug/endpoints/:key", auth, statusPageEndpointHandler(notFound))
		// Fork: notifications of the new results of an endpoint of the page in real time, see api/live_updates.go
		unprotectedAPIRouter.Get("/v1/status-pages/:slug/endpoints/:key/events", auth, statusPageEndpointEventsHandler(notFound, trustedProxies))
		// Fork: data of the response time chart of an endpoint of the page, see api/response_time_chart.go
		unprotectedAPIRouter.Get("/v1/status-pages/:slug/endpoints/:key/response-time-chart", auth, statusPageResponseTimeChartHandler(cfg, notFound))
		// Fork: badges of the page, so that a page with a login does not publish its numbers through the global routes
		unprotectedAPIRouter.Get("/v1/status-pages/:slug/endpoints/:key/health/badge.svg", auth, statusPageBadgeHandler(notFound, HealthBadge))
		unprotectedAPIRouter.Get("/v1/status-pages/:slug/endpoints/:key/response-times/:duration/badge.svg", auth, statusPageBadgeHandler(notFound, ResponseTimeBadge(cfg)))
	}
	unprotectedAPIRouter.All("/v1/status-pages", notFound)
	unprotectedAPIRouter.All("/v1/status-pages/*", notFound)
	// The HTML never reveals whether a page exists: it is always 200, and the page asks the API. A page with a login is
	// the exception: it answers 401 so that the browser asks for the credential.
	spa := renderSPA(cfg.UI, func(c *fiber.Ctx) {
		setPublicHeaders(c)
		// Fork: the HTML of a page that requires a login is private, like every other answer of that page
		if protected, _ := c.Locals(localsProtectedPageHTML).(bool); protected {
			setProtectedPageCacheControl(c, true, "")
		}
	})
	app.Get("/status/:slug", statusPageHTMLAuth(trustedProxies), spa)
	app.Get("/status/*", statusPageHTMLAuth(trustedProxies), spa)
}

func statusPageHandler(notFound fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		body, err := statuspage.PublicPageOf(c.Params("slug"), published)
		switch {
		case err == nil:
			setPublicAPIHeaders(c)
			// The payload is cached by the server: an HTTP cache must not keep a page that was just disabled
			setProtectedPageCacheControl(c, published.Page.RequiresLogin(), "no-cache")
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Status(fiber.StatusOK).Send(body)
		case errors.Is(err, statuspage.ErrPageNotFound):
			return notFound(c)
		default:
			return sendStatusPageError(c, fiber.StatusServiceUnavailable, statusPageUnavailableBody)
		}
	}
}

// statusPageEndpointHandler serves the details of an endpoint of a published status page, see
// statuspage.PublicEndpointDetails
func statusPageEndpointHandler(notFound fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil {
			return notFound(c)
		}
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		body, err := statuspage.PublicEndpointDetailsOf(c.Params("slug"), published, key)
		switch {
		case err == nil:
			setPublicAPIHeaders(c)
			setProtectedPageCacheControl(c, published.Page.RequiresLogin(), "no-cache")
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Status(fiber.StatusOK).Send(body)
		case errors.Is(err, statuspage.ErrPageNotFound):
			return notFound(c)
		default:
			return sendStatusPageError(c, fiber.StatusServiceUnavailable, statusPageUnavailableBody)
		}
	}
}

// statusPageNotFound answers the same 404 for every page that is not published and every path that is not a page. Only
// these responses count in the rate limit of the client.
func statusPageNotFound(trustedProxies []netip.Prefix) fiber.Handler {
	return func(c *fiber.Ctx) error {
		remoteIP, _ := netip.AddrFromSlice(c.Context().RemoteIP())
		forwardedFor := c.Request().Header.PeekAll(fiber.HeaderXForwardedFor)
		statuspage.ObserveConnection(remoteIP, len(forwardedFor) > 0, trustedProxies)
		clientIP := statuspage.ClientIP(remoteIP, forwardedFor, trustedProxies)
		if allowed, retryAfter := statuspage.HitLimiter(clientIP, time.Now()); !allowed {
			logr.Debugf("[api.statusPageNotFound] Too many requests from %s", clientIP)
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			return sendStatusPageError(c, fiber.StatusTooManyRequests, statusPageTooManyRequestsBody)
		}
		return sendStatusPageError(c, fiber.StatusNotFound, statusPageNotFoundBody)
	}
}

func sendStatusPageError(c *fiber.Ctx, status int, body string) error {
	setPublicAPIHeaders(c)
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Status(status).SendString(body)
}

// setPublicHeaders sets the headers of every public status page response
func setPublicHeaders(c *fiber.Ctx) {
	c.Set("X-Robots-Tag", "noindex, nofollow")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderReferrerPolicy, "strict-origin-when-cross-origin")
}

// setPublicAPIHeaders sets the headers of the public status page API, with Vary set whether or not the body is compressed
func setPublicAPIHeaders(c *fiber.Ctx) {
	setPublicHeaders(c)
	c.Vary(fiber.HeaderAcceptEncoding)
}
