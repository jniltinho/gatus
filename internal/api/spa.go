package api

import (
	"bytes"
	_ "embed"
	"html/template"

	"gatus/v5/internal/config/ui"
	"gatus/v5/internal/httpx"
	static "gatus/v5/web"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

// SinglePageApplication serves the single page application of the dashboard. The template is parsed once, when the
// router is created, and not on every request as it used to be.
func SinglePageApplication(uiConfig *ui.Config) echo.HandlerFunc {
	t, parseErr := template.ParseFS(static.FileSystem, static.IndexPath)
	return func(c *echo.Context) error {
		if parseErr != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.SinglePageApplication] Failed to parse template. This should never happen, because the template is validated on start. Error: %s", parseErr.Error())
			return httpx.SendString(c, 500, "Failed to parse template. This should never happen, because the template is validated on start.")
		}
		// Fork: the same theme rules as the public status pages, see themeFromRequest
		vd := ui.ViewData{UI: uiConfig, Theme: themeFromRequest(c, uiConfig), DefaultTheme: defaultTheme(uiConfig)}
		// Rendered into a buffer: once the first byte is written the answer cannot become an error anymore
		var body bytes.Buffer
		if err := t.Execute(&body, vd); err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.SinglePageApplication] Failed to execute template. This should never happen, because the template is validated on start. Error: %s", err.Error())
			return httpx.SendString(c, 500, "Failed to parse template. This should never happen, because the template is validated on start.")
		}
		httpx.SetHeader(c, "Content-Type", "text/html")
		return httpx.Send(c, 200, body.Bytes())
	}
}
