package api

import (
	"mime"
	"net/url"
	"os"
	"slices"
	"strings"

	"gatus/v5/config/admin"
	"github.com/gofiber/fiber/v2"
)

const (
	// adminMaximumBodySize is the maximum size of the body of requests to the administration API
	adminMaximumBodySize = 256 * 1024

	// adminDevServerOrigin is the origin of the Vue development server, accepted when ENVIRONMENT=dev
	adminDevServerOrigin = "http://localhost:8081"
)

var adminMediaTypes = []string{"application/json", "application/yaml", "application/x-yaml", "text/yaml"}

// adminRequestProtection protects the requests that change managed endpoints against CSRF and oversized bodies.
//
// The expected origin is derived only from the Host header and the scheme (TLS or X-Forwarded-Proto). Pages cannot
// set Host, Origin or Sec-Fetch-* and a cross-site request with a custom header such as X-Forwarded-Proto requires a
// CORS preflight that Gatus does not allow, so this is safe against CSRF. X-Forwarded-Host is ignored on purpose, and
// so are Fiber's Hostname() and Protocol(), which trust any X-Forwarded-* header.
func adminRequestProtection(adminConfig *admin.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		default:
			return c.Next()
		}
		if strings.EqualFold(c.Get("Sec-Fetch-Site"), "cross-site") {
			return adminError(c, fiber.StatusForbidden, "cross-site requests are not allowed")
		}
		if origin, present := requestOrigin(c); present && !isAllowedAdminOrigin(c, origin, adminConfig) {
			return adminError(c, fiber.StatusForbidden, "origin is not allowed: "+origin)
		}
		body := c.Body()
		// Fork: the restore routes accept a backup file, checked by their handler (see api/admin_backup.go)
		if len(body) > adminMaximumBodySize && !isAdminRestorePath(c.Path()) {
			return adminError(c, fiber.StatusRequestEntityTooLarge, "request body is too large")
		}
		if len(body) > 0 && !isAdminMediaType(c.Get(fiber.HeaderContentType)) {
			return adminError(c, fiber.StatusUnsupportedMediaType, "content type must be application/json or application/yaml")
		}
		return c.Next()
	}
}

func adminError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

// requestOrigin returns the Origin of the request or, without it, the origin of the Referer
func requestOrigin(c *fiber.Ctx) (string, bool) {
	if origin := strings.TrimSpace(c.Get(fiber.HeaderOrigin)); len(origin) > 0 {
		return normalizeOrigin(origin), true
	}
	if referer := strings.TrimSpace(c.Get(fiber.HeaderReferer)); len(referer) > 0 {
		parsed, err := url.Parse(referer)
		if err != nil || len(parsed.Scheme) == 0 || len(parsed.Host) == 0 {
			return referer, true
		}
		return normalizeOrigin(parsed.Scheme + "://" + parsed.Host), true
	}
	return "", false
}

func isAllowedAdminOrigin(c *fiber.Ctx, origin string, adminConfig *admin.Config) bool {
	if os.Getenv("ENVIRONMENT") == "dev" && origin == adminDevServerOrigin {
		return true
	}
	if adminConfig != nil && len(adminConfig.AllowedOrigins) > 0 {
		for _, allowedOrigin := range adminConfig.AllowedOrigins {
			if normalizeOrigin(allowedOrigin) == origin {
				return true
			}
		}
		return false
	}
	return origin == derivedOrigin(c)
}

// derivedOrigin returns the origin of the request from its Host header and scheme
func derivedOrigin(c *fiber.Ctx) string {
	scheme := "http"
	if c.Context().IsTLS() {
		scheme = "https"
	} else if forwardedProto, _, _ := strings.Cut(c.Get(fiber.HeaderXForwardedProto), ","); slices.Contains([]string{"http", "https"}, strings.ToLower(strings.TrimSpace(forwardedProto))) {
		scheme = strings.ToLower(strings.TrimSpace(forwardedProto))
	}
	return normalizeOrigin(scheme + "://" + string(c.Request().Host()))
}

// normalizeOrigin lowercases an origin and removes the default port of its scheme
func normalizeOrigin(origin string) string {
	origin = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(origin), "/"))
	if strings.HasPrefix(origin, "http://") {
		return strings.TrimSuffix(origin, ":80")
	}
	if strings.HasPrefix(origin, "https://") {
		return strings.TrimSuffix(origin, ":443")
	}
	return origin
}

func isAdminMediaType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && slices.Contains(adminMediaTypes, strings.ToLower(mediaType))
}
