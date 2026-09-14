package api

import (
	"fmt"
	"net/http"

	"gatus/v5/security"
	"gatus/v5/statuspage"
	"github.com/gofiber/fiber/v2"
)

type adminStatusPageHandler struct {
	service  *statuspage.Service
	security *security.Config
}

// registerAdminStatusPageRoutes registers the routes of the administration of the status pages on the administration
// router, which already requires authentication, administrator permission and request protection. The fixed segments
// are registered before /:slug, and the slugs matching them are reserved.
func registerAdminStatusPageRoutes(router fiber.Router, securityConfig *security.Config) {
	handler := &adminStatusPageHandler{service: statuspage.NewService(), security: securityConfig}
	router.Get("/status-pages", handler.list)
	router.Get("/status-pages/options", handler.options)
	router.Get("/status-pages/exposure", handler.exposure)
	router.Post("/status-pages/validate", handler.validate)
	router.Post("/status-pages", handler.create)
	router.Get("/status-pages/:slug", handler.get)
	router.Put("/status-pages/:slug", handler.update)
	router.Post("/status-pages/:slug/enable", handler.setEnabled(true))
	router.Post("/status-pages/:slug/disable", handler.setEnabled(false))
	router.Delete("/status-pages/:slug", handler.delete)
	router.Get("/status-pages/:slug/preview", handler.preview)
}

func (h *adminStatusPageHandler) list(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(h.service.List())
}

func (h *adminStatusPageHandler) options(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(h.service.Options())
}

func (h *adminStatusPageHandler) exposure(c *fiber.Ctx) error {
	exposure, err := h.service.Exposure(c.Query("group"), c.Query("key"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return c.Status(http.StatusOK).JSON(exposure)
}

func (h *adminStatusPageHandler) validate(c *fiber.Ctx) error {
	validation, err := h.service.Validate(c.Body(), c.Query("slug"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return c.Status(http.StatusOK).JSON(validation)
}

func (h *adminStatusPageHandler) create(c *fiber.Ctx) error {
	detail, err := h.service.Create(c.Body(), h.security.RequestAuthor(c))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return writeAdminStatusPageDetail(c, http.StatusCreated, detail)
}

func (h *adminStatusPageHandler) get(c *fiber.Ctx) error {
	detail, err := h.service.Get(c.Params("slug"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return writeAdminStatusPageDetail(c, http.StatusOK, detail)
}

func (h *adminStatusPageHandler) update(c *fiber.Ctx) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminStatusPageError(c, err)
	}
	detail, err := h.service.Update(c.Params("slug"), c.Body(), expectedVersion, h.security.RequestAuthor(c))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return writeAdminStatusPageDetail(c, http.StatusOK, detail)
}

func (h *adminStatusPageHandler) setEnabled(enabled bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		expectedVersion, err := adminExpectedVersion(c)
		if err != nil {
			return adminStatusPageError(c, err)
		}
		detail, err := h.service.SetEnabled(c.Params("slug"), enabled, expectedVersion, h.security.RequestAuthor(c))
		if err != nil {
			return adminStatusPageError(c, err)
		}
		return writeAdminStatusPageDetail(c, http.StatusOK, detail)
	}
}

func (h *adminStatusPageHandler) delete(c *fiber.Ctx) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminStatusPageError(c, err)
	}
	slug := c.Params("slug")
	if err := h.service.Delete(slug, expectedVersion, h.security.RequestAuthor(c)); err != nil {
		return adminStatusPageError(c, err)
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"slug": slug})
}

func (h *adminStatusPageHandler) preview(c *fiber.Ctx) error {
	body, err := h.service.Preview(c.Params("slug"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Status(http.StatusOK).Send(body)
}

func writeAdminStatusPageDetail(c *fiber.Ctx, status int, detail *statuspage.Detail) error {
	if detail.Version > 0 {
		c.Set(fiber.HeaderETag, fmt.Sprintf(`"%d"`, detail.Version))
	}
	return c.Status(status).JSON(detail)
}
