package api

import (
	"bytes"
	"html/template"

	"gatus/v5/config/ui"
	static "gatus/v5/web"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

// renderSPA returns a handler that always renders the single page application with status 200, from a template parsed
// once when the router is created (once per start or reload), and sets headers on every response
func renderSPA(uiConfig *ui.Config, headers func(c *fiber.Ctx)) fiber.Handler {
	indexTemplate, parseErr := template.ParseFS(static.FileSystem, static.IndexPath)
	return func(c *fiber.Ctx) error {
		if parseErr != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.renderSPA] Failed to parse template: %s", parseErr.Error())
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		var body bytes.Buffer
		if err := indexTemplate.Execute(&body, ui.ViewData{UI: uiConfig, Theme: themeFromRequest(c, uiConfig)}); err != nil {
			logr.Errorf("[api.renderSPA] Failed to execute template: %s", err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute template. This should never happen, because the template is validated on start.")
		}
		headers(c)
		// The template depends on the theme cookie
		c.Set(fiber.HeaderCacheControl, "no-cache")
		c.Set(fiber.HeaderContentType, "text/html")
		return c.Status(fiber.StatusOK).Send(body.Bytes())
	}
}

// themeFromRequest returns the theme of the theme cookie or, without cookie, the one configured in ui.dark-mode
func themeFromRequest(c *fiber.Ctx, uiConfig *ui.Config) string {
	if themeFromCookie := string(c.Request().Header.Cookie("theme")); len(themeFromCookie) > 0 {
		if themeFromCookie == "dark" {
			return "dark"
		}
		return ""
	}
	if uiConfig.IsDarkMode() {
		return "dark"
	}
	return ""
}
