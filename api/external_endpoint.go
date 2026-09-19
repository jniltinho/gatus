package api

import (
	"errors"
	"strings"
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/internal/httpx"
	"gatus/v5/storage/store/common"
	"gatus/v5/watchdog"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

func CreateExternalEndpointResult(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// Check if the success query parameter is present
		// The LAST occurrence is the one that is validated and the FIRST is the one that counts, exactly as before:
		// ?success=true&success=invalid is refused
		success, exists := httpx.QueryLast(c, "success")
		if !exists || (success != "true" && success != "false") {
			return httpx.SendString(c, 400, "missing or invalid success query parameter")
		}
		// Check if the authorization bearer token header is correct
		authorizationHeader := httpx.Header(c, echo.HeaderAuthorization)
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			return httpx.SendString(c, 401, "invalid Authorization header")
		}
		token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
		if len(token) == 0 {
			return httpx.SendString(c, 401, "bearer token must not be empty")
		}
		key := c.Param("key")
		externalEndpoint := cfg.GetExternalEndpointByKey(key)
		if externalEndpoint == nil {
			logr.Errorf("[api.CreateExternalEndpointResult] External endpoint with key=%s not found", key)
			return httpx.SendString(c, 404, "not found")
		}
		if externalEndpoint.Token != token {
			logr.Errorf("[api.CreateExternalEndpointResult] Invalid token for external endpoint with key=%s", key)
			return httpx.SendString(c, 401, "invalid token")
		}
		// Fork: a result of this API is reported by an external system, like a push: with the origin, a status page that
		// shows messages never turns the text sent in error= into a reason of its own
		result := &endpoint.Result{
			Timestamp: time.Now(),
			Success:   httpx.Query(c, "success") == "true",
			Errors:    []string{},
			Origin:    endpoint.ResultOriginPush,
		}
		if len(httpx.Query(c, "duration")) > 0 {
			parsedDuration, err := time.ParseDuration(httpx.Query(c, "duration"))
			if err != nil {
				logr.Errorf("[api.CreateExternalEndpointResult] Invalid duration from string=%s with error: %s", httpx.Query(c, "duration"), err.Error())
				return httpx.SendString(c, 400, "invalid duration: "+err.Error())
			}
			result.Duration = parsedDuration
		}
		if errorFromQuery := httpx.Query(c, "error"); !result.Success && len(errorFromQuery) > 0 {
			result.AddError(errorFromQuery)
		}
		// Fork: stores the result, publishes its metrics and handles its alerts one result at a time, like the pushes and
		// the heartbeat of the endpoint
		if err := watchdog.ProcessExternalEndpointResult(externalEndpoint, result, cfg, true); err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return httpx.SendString(c, 404, err.Error())
			}
			logr.Errorf("[api.CreateExternalEndpointResult] Failed to insert result in storage: %s", err.Error())
			return httpx.SendString(c, 500, err.Error())
		}
		logr.Infof("[api.CreateExternalEndpointResult] Successfully inserted result for external endpoint with key=%s and success=%s", c.Param("key"), success)
		// Return the result
		return httpx.SendString(c, 200, "")
	}
}
