package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gatus/v5/config"
	"gatus/v5/managedendpoint"
	"gatus/v5/security"
	"gatus/v5/storage/store/common"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

var errAdminVersionRequired = errors.New("the If-Match header with the current version of the endpoint is required")

type adminHandler struct {
	service  *managedendpoint.Service
	security *security.Config
}

// registerAdminRoutes registers the routes of the administration of endpoints. The router must already require
// authentication, administrator permission and request protection.
//
// Handlers always set the status explicitly: the static file middleware registered before the protected routes sets
// 404 when no file matches.
func registerAdminRoutes(router fiber.Router, cfg *config.Config) {
	handler := &adminHandler{service: managedendpoint.NewService(cfg), security: cfg.Security}
	router.Get("/metadata", handler.metadata)
	router.Get("/endpoints", handler.list)
	router.Post("/endpoints/parse", handler.parse)
	router.Post("/endpoints/validate", handler.validate)
	router.Post("/endpoints/test", handler.test)
	router.Post("/endpoints", handler.create)
	router.Get("/endpoints/:key", handler.get)
	router.Put("/endpoints/:key", handler.update)
	router.Post("/endpoints/:key/enable", handler.setEnabled(true))
	router.Post("/endpoints/:key/disable", handler.setEnabled(false))
	router.Delete("/endpoints/:key", handler.delete)
	// Status pages (see api/admin_status_pages.go)
	registerAdminStatusPageRoutes(router, cfg.Security)
}

func (h *adminHandler) metadata(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(h.service.Metadata())
}

func (h *adminHandler) list(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(h.service.List())
}

func (h *adminHandler) get(c *fiber.Ctx) error {
	detail, err := h.service.Get(adminKey(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	return writeAdminDetail(c, http.StatusOK, detail)
}

func (h *adminHandler) parse(c *fiber.Ctx) error {
	document, err := h.service.ParseDefinition(c.Body())
	if err != nil {
		return adminServiceError(c, err)
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"json": document})
}

func (h *adminHandler) validate(c *fiber.Ctx) error {
	validation, err := h.service.Validate(c.Body(), strings.ToLower(c.Query("key")))
	if err != nil {
		return adminServiceError(c, err)
	}
	return c.Status(http.StatusOK).JSON(validation)
}

func (h *adminHandler) test(c *fiber.Ctx) error {
	result, err := h.service.Test(c.Body(), strings.ToLower(c.Query("key")))
	if err != nil {
		return adminServiceError(c, err)
	}
	return c.Status(http.StatusOK).JSON(result)
}

func (h *adminHandler) create(c *fiber.Ctx) error {
	detail, err := h.service.Create(c.Body(), h.security.RequestAuthor(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	return writeAdminDetail(c, http.StatusCreated, detail)
}

func (h *adminHandler) update(c *fiber.Ctx) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminServiceError(c, err)
	}
	detail, err := h.service.Update(adminKey(c), c.Body(), expectedVersion, h.security.RequestAuthor(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	return writeAdminDetail(c, http.StatusOK, detail)
}

func (h *adminHandler) setEnabled(enabled bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		expectedVersion, err := adminExpectedVersion(c)
		if err != nil {
			return adminServiceError(c, err)
		}
		detail, err := h.service.SetEnabled(adminKey(c), enabled, expectedVersion, h.security.RequestAuthor(c))
		if err != nil {
			return adminServiceError(c, err)
		}
		return writeAdminDetail(c, http.StatusOK, detail)
	}
}

func (h *adminHandler) delete(c *fiber.Ctx) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminServiceError(c, err)
	}
	key := adminKey(c)
	triggeredAlerts, err := h.service.Delete(key, expectedVersion, h.security.RequestAuthor(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	// The endpoint must not show up in the cached statuses anymore
	_ = cache.DeleteKeysByPattern("endpoint-status-*")
	return c.Status(http.StatusOK).JSON(fiber.Map{"key": key, "triggeredAlerts": triggeredAlerts})
}

func adminKey(c *fiber.Ctx) string {
	key, err := url.PathUnescape(c.Params("key"))
	if err != nil {
		key = c.Params("key")
	}
	return strings.ToLower(key)
}

// adminExpectedVersion returns the version from the If-Match header, e.g. "3" or W/"3"
func adminExpectedVersion(c *fiber.Ctx) (int64, error) {
	value := strings.TrimSpace(c.Get(fiber.HeaderIfMatch))
	if len(value) == 0 {
		return 0, errAdminVersionRequired
	}
	value = strings.Trim(strings.TrimPrefix(value, "W/"), `"`)
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("%w: invalid If-Match header", common.ErrManagedEndpointVersionMismatch)
	}
	return version, nil
}

func writeAdminDetail(c *fiber.Ctx, status int, detail *managedendpoint.Detail) error {
	if detail.Version > 0 {
		c.Set(fiber.HeaderETag, fmt.Sprintf(`"%d"`, detail.Version))
	}
	return c.Status(status).JSON(detail)
}

func adminServiceError(c *fiber.Ctx, err error) error {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, managedendpoint.ErrNotFound), errors.Is(err, common.ErrManagedEndpointNotFound):
		status = http.StatusNotFound
	case errors.Is(err, managedendpoint.ErrReadOnly), errors.Is(err, managedendpoint.ErrKeyConflict), errors.Is(err, common.ErrManagedEndpointAlreadyExists):
		status = http.StatusConflict
	case errors.Is(err, common.ErrManagedEndpointVersionMismatch):
		status = http.StatusPreconditionFailed
	case errors.Is(err, errAdminVersionRequired):
		status = http.StatusPreconditionRequired
	case errors.Is(err, managedendpoint.ErrTooManyTests):
		status = http.StatusTooManyRequests
	case errors.Is(err, managedendpoint.ErrCycleInProgress):
		status = http.StatusServiceUnavailable
	case errors.Is(err, managedendpoint.ErrEmptyDefinition), errors.Is(err, managedendpoint.ErrInvalidDefinition),
		errors.Is(err, managedendpoint.ErrFieldNotAllowed), errors.Is(err, managedendpoint.ErrAlertProviderNotConfigured),
		errors.Is(err, managedendpoint.ErrInvalidAlertOverride), errors.Is(err, managedendpoint.ErrExtraLabelNotAllowed),
		errors.Is(err, managedendpoint.ErrKeyChanged):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		logr.Errorf("[api.adminServiceError] %s", err.Error())
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}
