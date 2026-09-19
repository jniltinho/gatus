package api

import (
	"encoding/json"
	"fmt"

	"gatus/v5/internal/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/internal/security"

	"github.com/labstack/echo/v5"
)

type ConfigHandler struct {
	securityConfig *security.Config
	config         *config.Config
}

func (handler ConfigHandler) GetConfig(c *echo.Context) error {
	hasOIDC := false
	login := ""
	isAuthenticated := true // Default to true if no security config is set
	if handler.securityConfig != nil {
		hasOIDC = handler.securityConfig.OIDC != nil
		// Login method of the frontend (fork): "basic" shows the login screen, "oidc" the OIDC login
		login = handler.securityConfig.LoginMethod()
		// With security.basic, the request is authenticated once: IsAdmin reuses this result, so that a wrong password
		// counts as a single failure and is checked once
		isAuthenticated = handler.securityConfig.IsAuthenticated(c)
	}

	// Prepare response with announcements
	response := map[string]interface{}{
		"oidc":          hasOIDC,
		"login":         login,
		"authenticated": isAuthenticated,
	}
	// Administration of endpoints (fork): whether it is enabled and whether the current request can use it
	if handler.config != nil {
		response["admin"] = map[string]bool{
			"enabled":    handler.config.Admin.IsEnabled(),
			"authorized": handler.securityConfig.IsAdmin(c, handler.config.Admin),
		}
	}
	// Add announcements if available, otherwise use empty slice
	if handler.config != nil && handler.config.Announcements != nil && len(handler.config.Announcements) > 0 {
		response["announcements"] = handler.config.Announcements
	} else {
		response["announcements"] = []interface{}{}
	}

	// Return the config as JSON
	httpx.SetHeader(c, "Content-Type", "application/json")
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return httpx.SendString(c, 500, fmt.Sprintf(`{"error":"Failed to marshal response: %s"}`, err.Error()))
	}
	return httpx.Send(c, 200, responseBytes)
}
