package api

import (
	_ "embed"
	"html/template"

	"gatus/v5/config/ui"
	static "gatus/v5/web"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

func SinglePageApplication(uiConfig *ui.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fork: the same theme rules as the public status pages, see themeFromRequest
		vd := ui.ViewData{UI: uiConfig, Theme: themeFromRequest(c, uiConfig), DefaultTheme: defaultTheme(uiConfig)}
		t, err := template.ParseFS(static.FileSystem, static.IndexPath)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.SinglePageApplication] Failed to parse template. This should never happen, because the template is validated on start. Error: %s", err.Error())
			return c.Status(500).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		c.Set("Content-Type", "text/html")
		err = t.Execute(c, vd)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			logr.Errorf("[api.SinglePageApplication] Failed to execute template. This should never happen, because the template is validated on start. Error: %s", err.Error())
			return c.Status(500).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		return c.SendStatus(200)
	}
}
