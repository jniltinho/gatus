package api

import (
	"encoding/base64"
	"math"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gatus/v5/statuspage"
	"github.com/gofiber/fiber/v2"
)

// Fork: login of a status page. The middleware runs before every route of a page and does, in this order:
//
//  1. resolves the slug once and keeps the published page in the locals, so that the route that answers uses exactly
//     the definition that was authorised;
//  2. answers the 404 of a page that does not exist, is not published or is not a page, exactly as before and without
//     WWW-Authenticate, so that nothing new is revealed;
//  3. challenges with 401 when the page requires a login, always sending WWW-Authenticate — including to requests that
//     look like they come from a browser, which is the opposite of what the login screen of security.basic needs.
//
// The 404 of a key that does not belong to the page stays in the route, after the challenge, so that the answer does
// not tell which endpoints a protected page has.
const (
	// localsPublishedStatusPage is the key of the locals holding the status page captured by the middleware
	localsPublishedStatusPage = "gatus.status-page"

	statusPageUnauthorizedBody = `{"error":"authentication required"}`

	// localsProtectedEventStream marks the event stream of a page that requires a login
	localsProtectedEventStream = "gatus.status-page-protected-stream"

	// localsProtectedPageHTML marks the HTML of a page that requires a login
	localsProtectedPageHTML = "gatus.status-page-protected-html"
)

// statusPageAuth returns the middleware of the routes of a status page
func statusPageAuth(notFound fiber.Handler, trustedProxies []netip.Prefix) fiber.Handler {
	return func(c *fiber.Ctx) error {
		published, ok := statuspage.Lookup(c.Params("slug"))
		if !ok {
			return notFound(c)
		}
		c.Locals(localsPublishedStatusPage, published)
		if !published.Page.RequiresLogin() {
			return c.Next()
		}
		username, password, hasCredential := basicCredentials(c)
		clientIP := statusPageClientIP(c, trustedProxies)
		result, retryAfter := statuspage.VerifyPageCredential(published.Page, published.Page.Slug, username, password, hasCredential, clientIP, time.Now())
		switch result {
		case statuspage.AuthTooManyFailures:
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			return sendStatusPageError(c, fiber.StatusTooManyRequests, statusPageTooManyRequestsBody)
		case statuspage.AuthUnauthorized:
			return sendStatusPageUnauthorized(c, published.Page.Slug)
		default:
			return c.Next()
		}
	}
}

// publishedStatusPage returns the page captured by the middleware, and false when the route ran without it
func publishedStatusPage(c *fiber.Ctx) (statuspage.Published, bool) {
	published, ok := c.Locals(localsPublishedStatusPage).(statuspage.Published)
	return published, ok
}

// sendStatusPageUnauthorized answers the challenge of a page. The realm is the slug of the published definition, which
// is validated, and never the parameter of the request.
func sendStatusPageUnauthorized(c *fiber.Ctx, slug string) error {
	c.Set(fiber.HeaderWWWAuthenticate, `Basic realm="`+slug+`", charset="UTF-8"`)
	return sendStatusPageError(c, fiber.StatusUnauthorized, statusPageUnauthorizedBody)
}

// setProtectedPageCacheControl keeps the answers of a protected page out of any shared cache
func setProtectedPageCacheControl(c *fiber.Ctx, protected bool, public string) {
	if protected {
		c.Set(fiber.HeaderCacheControl, "private, no-store")
		c.Vary(fiber.HeaderAuthorization)
		return
	}
	c.Set(fiber.HeaderCacheControl, public)
}

// statusPageClientIP resolves the client of a request with status-pages.trusted-proxies
func statusPageClientIP(c *fiber.Ctx, trustedProxies []netip.Prefix) netip.Addr {
	remoteIP, _ := netip.AddrFromSlice(c.Context().RemoteIP())
	return statuspage.ClientIP(remoteIP, c.Request().Header.PeekAll(fiber.HeaderXForwardedFor), trustedProxies)
}

// basicCredentials returns the username and the password of the Authorization: Basic header of the request
func basicCredentials(c *fiber.Ctx) (string, string, bool) {
	scheme, encoded, found := strings.Cut(strings.TrimSpace(c.Get(fiber.HeaderAuthorization)), " ")
	if !found || !strings.EqualFold(scheme, "basic") {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", "", false
	}
	username, password, found := strings.Cut(string(decoded), ":")
	if !found {
		return "", "", false
	}
	return username, password, true
}

// statusPageBadgeHandler serves a badge of an endpoint of a page, with the same rules of the other routes of the page:
// the key that does not belong to the page answers 404 only after the challenge of the middleware
func statusPageBadgeHandler(notFound fiber.Handler, badge fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil || !statuspage.IsEndpointShownOf(published, key) {
			return notFound(c)
		}
		setPublicHeaders(c)
		if err = badge(c); err != nil {
			return err
		}
		if published.Page.RequiresLogin() {
			c.Set(fiber.HeaderCacheControl, "private, no-store")
			c.Vary(fiber.HeaderAuthorization)
		}
		return nil
	}
}

// statusPageHTMLAuth challenges the HTML routes of a page that requires a login. Every other path under /status/ keeps
// answering 200 with the HTML of the SPA, revealing nothing.
func statusPageHTMLAuth(trustedProxies []netip.Prefix) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := firstPathSegmentAfterStatus(c.Path())
		if len(slug) == 0 {
			return c.Next()
		}
		published, ok := statuspage.Lookup(slug)
		if !ok || !published.Page.RequiresLogin() {
			return c.Next()
		}
		username, password, hasCredential := basicCredentials(c)
		clientIP := statusPageClientIP(c, trustedProxies)
		result, retryAfter := statuspage.VerifyPageCredential(published.Page, published.Page.Slug, username, password, hasCredential, clientIP, time.Now())
		switch result {
		case statuspage.AuthTooManyFailures:
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			return sendStatusPageError(c, fiber.StatusTooManyRequests, statusPageTooManyRequestsBody)
		case statuspage.AuthUnauthorized:
			return sendStatusPageUnauthorized(c, published.Page.Slug)
		default:
			c.Locals(localsProtectedPageHTML, true)
			return c.Next()
		}
	}
}

// firstPathSegmentAfterStatus returns the slug of /status/<slug> and of any path under it, unescaped, or an empty
// string when the path has no slug
func firstPathSegmentAfterStatus(path string) string {
	rest, found := strings.CutPrefix(path, "/status/")
	if !found {
		return ""
	}
	if index := strings.IndexByte(rest, '/'); index >= 0 {
		rest = rest[:index]
	}
	slug, err := url.QueryUnescape(rest)
	if err != nil {
		return ""
	}
	return slug
}
