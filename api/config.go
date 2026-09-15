package api

import (
	"encoding/json"
	"fmt"

	"gatus/v5/config"
	"gatus/v5/security"
	"github.com/gofiber/fiber/v2"
)

type ConfigHandler struct {
	securityConfig *security.Config
	config         *config.Config
}

func (handler ConfigHandler) GetConfig(c *fiber.Ctx) error {
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
	c.Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return c.Status(500).SendString(fmt.Sprintf(`{"error":"Failed to marshal response: %s"}`, err.Error()))
	}
	return c.Status(200).Send(responseBytes)
}
