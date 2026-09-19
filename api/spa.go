package api

import (
	"bytes"
	_ "embed"
	"html/template"

	"gatus/v5/config/ui"
	"gatus/v5/internal/httpx"
	static "gatus/v5/web"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

func SinglePageApplication(uiConfig *ui.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// Fork: the same theme rules as the public status pages, see themeFromRequest
		vd := ui.ViewData{UI: uiConfig, Theme: themeFromRequest(c, uiConfig), DefaultTheme: defaultTheme(uiConfig)}
		t, err := template.ParseFS(static.FileSystem, static.IndexPath)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.SinglePageApplication] Failed to parse template. This should never happen, because the template is validated on start. Error: %s", err.Error())
			return httpx.SendString(c, 500, "Failed to parse template. This should never happen, because the template is validated on start.")
		}
		// Rendered into a buffer: once the first byte is written the answer cannot become an error anymore
		var body bytes.Buffer
		err = t.Execute(&body, vd)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.SinglePageApplication] Failed to execute template. This should never happen, because the template is validated on start. Error: %s", err.Error())
			return httpx.SendString(c, 500, "Failed to parse template. This should never happen, because the template is validated on start.")
		}
		httpx.SetHeader(c, "Content-Type", "text/html")
		return httpx.Send(c, 200, body.Bytes())
	}
}
