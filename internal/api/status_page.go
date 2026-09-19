package api

import (
	"errors"
	"math"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"gatus/v5/internal/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/internal/statuspage"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
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
func registerStatusPageRoutes(app httpx.Router, unprotectedAPIRouter httpx.Router, cfg *config.Config) {
	statuspage.ConfigureLimiter(cfg.StatusPages.GetRateLimit())
	trustedProxies := cfg.StatusPages.TrustedProxyPrefixes()
	notFound := statusPageNotFound(trustedProxies)
	// Fork: the middleware of the login of a page runs on each route of a page, never on the catch-all, and captures the
	// page for the route that answers
	auth := statusPageAuth(notFound, trustedProxies)
	if cfg.StatusPages.IsEnabled() {
		httpx.GetAndHead(unprotectedAPIRouter, "/v1/status-pages/:slug", statusPageHandler(notFound), auth)
		httpx.GetAndHead(unprotectedAPIRouter, "/v1/status-pages/:slug/endpoints/:key", statusPageEndpointHandler(notFound), auth)
		// Fork: notifications of the new results of an endpoint of the page in real time, see api/live_updates.go
		httpx.GetAndHead(unprotectedAPIRouter, "/v1/status-pages/:slug/endpoints/:key/events", statusPageEndpointEventsHandler(notFound, trustedProxies), auth)
		// Fork: data of the response time chart of an endpoint of the page, see api/response_time_chart.go
		httpx.GetAndHead(unprotectedAPIRouter, "/v1/status-pages/:slug/endpoints/:key/response-time-chart", statusPageResponseTimeChartHandler(cfg, notFound), auth)
		// Fork: badges of the page, so that a page with a login does not publish its numbers through the global routes
		httpx.GetAndHead(unprotectedAPIRouter, "/v1/status-pages/:slug/endpoints/:key/health/badge.svg", statusPageBadgeHandler(notFound, HealthBadge), auth)
		httpx.GetAndHead(unprotectedAPIRouter, "/v1/status-pages/:slug/endpoints/:key/response-times/:duration/badge.svg", statusPageBadgeHandler(notFound, ResponseTimeBadge(cfg)), auth)
	}
	unprotectedAPIRouter.Any("/v1/status-pages", notFound)
	unprotectedAPIRouter.Any("/v1/status-pages/*", notFound)
	// The HTML never reveals whether a page exists: it is always 200, and the page asks the API. A page with a login is
	// the exception: it answers 401 so that the browser asks for the credential.
	spa := renderSPA(cfg.UI, func(c *echo.Context) {
		setPublicHeaders(c)
		// Fork: the HTML of a page that requires a login is private, like every other answer of that page
		if protected, _ := c.Get(localsProtectedPageHTML).(bool); protected {
			setProtectedPageCacheControl(c, true, "")
		}
	})
	httpx.GetAndHead(app, "/status/:slug", spa, statusPageHTMLAuth(trustedProxies))
	httpx.GetAndHead(app, "/status/*", spa, statusPageHTMLAuth(trustedProxies))
}

func statusPageHandler(notFound echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		body, err := statuspage.PublicPageOf(c.Param("slug"), published)
		switch {
		case err == nil:
			setPublicAPIHeaders(c)
			// The payload is cached by the server: an HTTP cache must not keep a page that was just disabled
			setProtectedPageCacheControl(c, published.Page.RequiresLogin(), "no-cache")
			httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
			return httpx.Send(c, http.StatusOK, body)
		case errors.Is(err, statuspage.ErrPageNotFound):
			return notFound(c)
		default:
			return sendStatusPageError(c, http.StatusServiceUnavailable, statusPageUnavailableBody)
		}
	}
}

// statusPageEndpointHandler serves the details of an endpoint of a published status page, see
// statuspage.PublicEndpointDetails
func statusPageEndpointHandler(notFound echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil {
			return notFound(c)
		}
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		body, err := statuspage.PublicEndpointDetailsOf(c.Param("slug"), published, key)
		switch {
		case err == nil:
			setPublicAPIHeaders(c)
			setProtectedPageCacheControl(c, published.Page.RequiresLogin(), "no-cache")
			httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
			return httpx.Send(c, http.StatusOK, body)
		case errors.Is(err, statuspage.ErrPageNotFound):
			return notFound(c)
		default:
			return sendStatusPageError(c, http.StatusServiceUnavailable, statusPageUnavailableBody)
		}
	}
}

// statusPageNotFound answers the same 404 for every page that is not published and every path that is not a page. Only
// these responses count in the rate limit of the client.
func statusPageNotFound(trustedProxies []netip.Prefix) echo.HandlerFunc {
	return func(c *echo.Context) error {
		remoteIP := httpx.RemoteIP(c)
		forwardedFor := httpx.HeaderValues(c, echo.HeaderXForwardedFor)
		statuspage.ObserveConnection(remoteIP, len(forwardedFor) > 0, trustedProxies)
		clientIP := statuspage.ClientIP(remoteIP, forwardedFor, trustedProxies)
		if allowed, retryAfter := statuspage.HitLimiter(clientIP, time.Now()); !allowed {
			logr.Debugf("[api.statusPageNotFound] Too many requests from %s", clientIP)
			httpx.SetHeader(c, echo.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			return sendStatusPageError(c, http.StatusTooManyRequests, statusPageTooManyRequestsBody)
		}
		return sendStatusPageError(c, http.StatusNotFound, statusPageNotFoundBody)
	}
}

func sendStatusPageError(c *echo.Context, status int, body string) error {
	setPublicAPIHeaders(c)
	httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
	httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
	return httpx.SendString(c, status, body)
}

// setPublicHeaders sets the headers of every public status page response
func setPublicHeaders(c *echo.Context) {
	httpx.SetHeader(c, "X-Robots-Tag", "noindex, nofollow")
	httpx.SetHeader(c, echo.HeaderXContentTypeOptions, "nosniff")
	httpx.SetHeader(c, echo.HeaderReferrerPolicy, "strict-origin-when-cross-origin")
}

// setPublicAPIHeaders sets the headers of the public status page API, with Vary set whether or not the body is compressed
func setPublicAPIHeaders(c *echo.Context) {
	setPublicHeaders(c)
	httpx.Vary(c, echo.HeaderAcceptEncoding)
}
