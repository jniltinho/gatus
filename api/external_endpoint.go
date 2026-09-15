package api

import (
	"errors"
	"strings"
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/storage/store/common"
	"gatus/v5/watchdog"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

func CreateExternalEndpointResult(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if the success query parameter is present
		success, exists := c.Queries()["success"]
		if !exists || (success != "true" && success != "false") {
			return c.Status(400).SendString("missing or invalid success query parameter")
		}
		// Check if the authorization bearer token header is correct
		authorizationHeader := string(c.Request().Header.Peek("Authorization"))
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			return c.Status(401).SendString("invalid Authorization header")
		}
		token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
		if len(token) == 0 {
			return c.Status(401).SendString("bearer token must not be empty")
		}
		key := c.Params("key")
		externalEndpoint := cfg.GetExternalEndpointByKey(key)
		if externalEndpoint == nil {
			logr.Errorf("[api.CreateExternalEndpointResult] External endpoint with key=%s not found", key)
			return c.Status(404).SendString("not found")
		}
		if externalEndpoint.Token != token {
			logr.Errorf("[api.CreateExternalEndpointResult] Invalid token for external endpoint with key=%s", key)
			return c.Status(401).SendString("invalid token")
		}
		result := &endpoint.Result{
			Timestamp: time.Now(),
			Success:   c.QueryBool("success"),
			Errors:    []string{},
		}
		if len(c.Query("duration")) > 0 {
			parsedDuration, err := time.ParseDuration(c.Query("duration"))
			if err != nil {
				logr.Errorf("[api.CreateExternalEndpointResult] Invalid duration from string=%s with error: %s", c.Query("duration"), err.Error())
				return c.Status(400).SendString("invalid duration: " + err.Error())
			}
			result.Duration = parsedDuration
		}
		if errorFromQuery := c.Query("error"); !result.Success && len(errorFromQuery) > 0 {
			result.AddError(errorFromQuery)
		}
		// Fork: stores the result, publishes its metrics and handles its alerts one result at a time, like the pushes and
		// the heartbeat of the endpoint
		if err := watchdog.ProcessExternalEndpointResult(externalEndpoint, result, cfg, true); err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			logr.Errorf("[api.CreateExternalEndpointResult] Failed to insert result in storage: %s", err.Error())
			return c.Status(500).SendString(err.Error())
		}
		logr.Infof("[api.CreateExternalEndpointResult] Successfully inserted result for external endpoint with key=%s and success=%s", c.Params("key"), success)
		// Return the result
		return c.Status(200).SendString("")
	}
}
