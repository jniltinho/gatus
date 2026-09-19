package api

import (
	"encoding/json"
	"errors"
	"math"
	"mime"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"gatus/v5/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/security"
	"gatus/v5/statuspage"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

// authMaximumBodySize is the maximum size of the body of the login requests
const authMaximumBodySize = 4 * 1024

// clientIPMiddleware keeps the IP address of the client, computed with status-pages.trusted-proxies, in the locals of
// the request for the failure limiter of security.basic (fork). Like the public status pages, it logs once per
// generation a warning when the connection looks like a reverse proxy that is not in status-pages.trusted-proxies.
func clientIPMiddleware(trustedProxies []netip.Prefix) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			remoteIP := httpx.RemoteIP(c)
			forwardedFor := httpx.HeaderValues(c, echo.HeaderXForwardedFor)
			statuspage.ObserveConnection(remoteIP, len(forwardedFor) > 0, trustedProxies)
			c.Set(security.LocalsClientIP, statuspage.ClientIP(remoteIP, forwardedFor, trustedProxies))
			return next(c)
		}
	}
}

// registerAuthRoutes registers the login and logout routes of the login screen of security.basic (fork). They must be
// registered before the security middleware, and respond with 404 without security.basic or with OIDC.
func registerAuthRoutes(unprotectedAPIRouter httpx.Router, cfg *config.Config, clientIP echo.MiddlewareFunc) {
	unprotectedAPIRouter.POST("/v1/auth/login", login(cfg), authRequestProtection(cfg), clientIP)
	unprotectedAPIRouter.POST("/v1/auth/logout", logout(cfg), authRequestProtection(cfg), clientIP)
}

// authRequestProtection applies the origin rules of the administration to the login and logout requests, against CSRF
// and login CSRF, see adminRequestProtection
func authRequestProtection(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
			if !cfg.Security.UsesBasicLogin() {
				return adminError(c, http.StatusNotFound, "not found")
			}
			if strings.EqualFold(httpx.Header(c, "Sec-Fetch-Site"), "cross-site") {
				return adminError(c, http.StatusForbidden, "cross-site requests are not allowed")
			}
			if origin, present := requestOrigin(c); present && !isAllowedAdminOrigin(c, origin, cfg.Admin) {
				return adminError(c, http.StatusForbidden, "origin is not allowed: "+origin)
			}
			return next(c)
		}
	}
}

// login creates a login session from the JSON credentials of the login screen. Neither the password nor the token is
// logged.
func login(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		body := httpx.Body(c)
		if len(body) > authMaximumBodySize {
			return adminError(c, http.StatusRequestEntityTooLarge, "request body is too large")
		}
		if mediaType, _, err := mime.ParseMediaType(httpx.Header(c, echo.HeaderContentType)); err != nil || !strings.EqualFold(mediaType, echo.MIMEApplicationJSON) {
			return adminError(c, http.StatusUnsupportedMediaType, "content type must be application/json")
		}
		var credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(body, &credentials); err != nil {
			return adminError(c, http.StatusBadRequest, "invalid request body")
		}
		clientIP := c.Get(security.LocalsClientIP)
		err := cfg.Security.Login(c, credentials.Username, credentials.Password)
		var tooManyFailures *security.TooManyFailuresError
		switch {
		case err == nil:
			logr.Infof("[api.login] Successful login from %v", clientIP)
			return httpx.SendStatus(c, http.StatusNoContent)
		case errors.As(err, &tooManyFailures):
			logr.Warnf("[api.login] Login refused for %v: too many failed attempts", clientIP)
			httpx.SetHeader(c, echo.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(tooManyFailures.RetryAfter.Seconds()))))
			return adminError(c, http.StatusTooManyRequests, "Too many failed login attempts, try again later")
		case errors.Is(err, security.ErrInvalidCredentials):
			logr.Warnf("[api.login] Failed login from %v", clientIP)
			return adminError(c, http.StatusUnauthorized, "Invalid username or password")
		default:
			logr.Errorf("[api.login] Failed to create the login session for %v: %s", clientIP, err.Error())
			return adminError(c, http.StatusInternalServerError, "failed to create the login session")
		}
	}
}

// logout deletes the login session of the request, if any, and expires its cookie
func logout(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		clientIP := c.Get(security.LocalsClientIP)
		if err := cfg.Security.Logout(c); err != nil {
			logr.Errorf("[api.logout] Failed to delete the login session for %v: %s", clientIP, err.Error())
			return adminError(c, http.StatusInternalServerError, "failed to delete the login session")
		}
		logr.Infof("[api.logout] Logout from %v", clientIP)
		return httpx.SendStatus(c, http.StatusNoContent)
	}
}
