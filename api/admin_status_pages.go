package api

import (
	"fmt"
	"net/http"

	"gatus/v5/internal/httpx"
	"gatus/v5/security"
	"gatus/v5/statuspage"

	"github.com/labstack/echo/v5"
)

type adminStatusPageHandler struct {
	service  *statuspage.Service
	security *security.Config
}

// registerAdminStatusPageRoutes registers the routes of the administration of the status pages on the administration
// router, which already requires authentication, administrator permission and request protection. The fixed segments
// are registered before /:slug, and the slugs matching them are reserved.
func registerAdminStatusPageRoutes(router httpx.Router, securityConfig *security.Config) {
	handler := &adminStatusPageHandler{service: statuspage.NewService(), security: securityConfig}
	httpx.GetAndHead(router, "/status-pages", handler.list)
	httpx.GetAndHead(router, "/status-pages/options", handler.options)
	httpx.GetAndHead(router, "/status-pages/exposure", handler.exposure)
	router.POST("/status-pages/validate", handler.validate)
	router.POST("/status-pages", handler.create)
	httpx.GetAndHead(router, "/status-pages/:slug", handler.get)
	router.PUT("/status-pages/:slug", handler.update)
	router.POST("/status-pages/:slug/enable", handler.setEnabled(true))
	router.POST("/status-pages/:slug/disable", handler.setEnabled(false))
	router.DELETE("/status-pages/:slug", handler.delete)
	httpx.GetAndHead(router, "/status-pages/:slug/preview", handler.preview)
}

func (h *adminStatusPageHandler) list(c *echo.Context) error {
	return httpx.JSON(c, http.StatusOK, h.service.List())
}

func (h *adminStatusPageHandler) options(c *echo.Context) error {
	return httpx.JSON(c, http.StatusOK, h.service.Options())
}

func (h *adminStatusPageHandler) exposure(c *echo.Context) error {
	exposure, err := h.service.Exposure(c.QueryParam("group"), c.QueryParam("key"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, exposure)
}

func (h *adminStatusPageHandler) validate(c *echo.Context) error {
	validation, err := h.service.Validate(httpx.Body(c), c.QueryParam("slug"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, validation)
}

func (h *adminStatusPageHandler) create(c *echo.Context) error {
	detail, err := h.service.Create(httpx.Body(c), h.security.RequestAuthor(c))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return writeAdminStatusPageDetail(c, http.StatusCreated, detail)
}

func (h *adminStatusPageHandler) get(c *echo.Context) error {
	detail, err := h.service.Get(c.Param("slug"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return writeAdminStatusPageDetail(c, http.StatusOK, detail)
}

func (h *adminStatusPageHandler) update(c *echo.Context) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminStatusPageError(c, err)
	}
	detail, err := h.service.Update(c.Param("slug"), httpx.Body(c), expectedVersion, h.security.RequestAuthor(c))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	return writeAdminStatusPageDetail(c, http.StatusOK, detail)
}

func (h *adminStatusPageHandler) setEnabled(enabled bool) echo.HandlerFunc {
	return func(c *echo.Context) error {
		expectedVersion, err := adminExpectedVersion(c)
		if err != nil {
			return adminStatusPageError(c, err)
		}
		detail, err := h.service.SetEnabled(c.Param("slug"), enabled, expectedVersion, h.security.RequestAuthor(c))
		if err != nil {
			return adminStatusPageError(c, err)
		}
		return writeAdminStatusPageDetail(c, http.StatusOK, detail)
	}
}

func (h *adminStatusPageHandler) delete(c *echo.Context) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminStatusPageError(c, err)
	}
	slug := c.Param("slug")
	if err := h.service.Delete(slug, expectedVersion, h.security.RequestAuthor(c)); err != nil {
		return adminStatusPageError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, map[string]any{"slug": slug})
}

func (h *adminStatusPageHandler) preview(c *echo.Context) error {
	body, err := h.service.Preview(c.Param("slug"))
	if err != nil {
		return adminStatusPageError(c, err)
	}
	httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
	httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
	return httpx.Send(c, http.StatusOK, body)
}

func writeAdminStatusPageDetail(c *echo.Context, status int, detail *statuspage.Detail) error {
	if detail.Version > 0 {
		httpx.SetHeader(c, "ETag", fmt.Sprintf(`"%d"`, detail.Version))
	}
	return httpx.JSON(c, status, detail)
}
