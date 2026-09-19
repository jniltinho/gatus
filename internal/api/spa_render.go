package api

import (
	"bytes"
	"html/template"
	"net/http"

	"gatus/v5/internal/config/ui"
	"gatus/v5/internal/httpx"
	static "gatus/v5/web"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

// renderSPA returns a handler that always renders the single page application with status 200, from a template parsed
// once when the router is created (once per start or reload), and sets headers on every response
func renderSPA(uiConfig *ui.Config, headers func(c *echo.Context)) echo.HandlerFunc {
	indexTemplate, parseErr := template.ParseFS(static.FileSystem, static.IndexPath)
	return func(c *echo.Context) error {
		if parseErr != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.renderSPA] Failed to parse template: %s", parseErr.Error())
			return httpx.SendString(c, http.StatusInternalServerError, "Failed to parse template. This should never happen, because the template is validated on start.")
		}
		var body bytes.Buffer
		if err := indexTemplate.Execute(&body, ui.ViewData{UI: uiConfig, Theme: themeFromRequest(c, uiConfig), DefaultTheme: defaultTheme(uiConfig)}); err != nil {
			logr.Errorf("[api.renderSPA] Failed to execute template: %s", err.Error())
			return httpx.SendString(c, http.StatusInternalServerError, "Failed to execute template. This should never happen, because the template is validated on start.")
		}
		// The template depends on the theme cookie. The headers of the route come after, so that a page that requires a
		// login can keep its HTML out of any shared cache (fork).
		httpx.SetHeader(c, echo.HeaderCacheControl, "no-cache")
		httpx.SetHeader(c, echo.HeaderContentType, "text/html")
		headers(c)
		return httpx.Send(c, http.StatusOK, body.Bytes())
	}
}

// themeFromRequest returns the theme of a valid theme cookie (dark or light) or, without one, the theme of ui.dark-mode.
// Fork: an invalid cookie is ignored, like in the browser (see web/app/src/utils/theme.js).
func themeFromRequest(c *echo.Context, uiConfig *ui.Config) string {
	var theme string
	if cookie, err := c.Cookie("theme"); err == nil {
		theme = cookie.Value
	}
	switch theme {
	case "dark":
		return "dark"
	case "light":
		return ""
	}
	return defaultTheme(uiConfig)
}

// defaultTheme returns the theme configured in ui.dark-mode, dark by default
func defaultTheme(uiConfig *ui.Config) string {
	if uiConfig.IsDarkMode() {
		return "dark"
	}
	return ""
}
