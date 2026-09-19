package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gatus/v5/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/managedendpoint"
	"gatus/v5/security"
	"gatus/v5/storage/store/common"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
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
func registerAdminRoutes(router httpx.Router, cfg *config.Config) {
	handler := &adminHandler{service: managedendpoint.NewService(cfg), security: cfg.Security}
	httpx.GetAndHead(router, "/metadata", handler.metadata)
	httpx.GetAndHead(router, "/endpoints", handler.list)
	router.POST("/endpoints/parse", handler.parse)
	router.POST("/endpoints/validate", handler.validate)
	router.POST("/endpoints/test", handler.test)
	router.POST("/endpoints", handler.create)
	httpx.GetAndHead(router, "/endpoints/:key", handler.get)
	router.PUT("/endpoints/:key", handler.update)
	router.POST("/endpoints/:key/enable", handler.setEnabled(true))
	router.POST("/endpoints/:key/disable", handler.setEnabled(false))
	router.DELETE("/endpoints/:key", handler.delete)
	// Status pages (see api/admin_status_pages.go)
	registerAdminStatusPageRoutes(router, cfg.Security)
	// Global push keys (see api/admin_push_keys.go)
	registerAdminPushKeyRoutes(router, cfg.Security)
	// Backup and restore (fork, see api/admin_backup.go)
	registerAdminBackupRoutes(router, cfg)
}

func (h *adminHandler) metadata(c *echo.Context) error {
	return httpx.JSON(c, http.StatusOK, h.service.Metadata())
}

func (h *adminHandler) list(c *echo.Context) error {
	return httpx.JSON(c, http.StatusOK, h.service.List())
}

func (h *adminHandler) get(c *echo.Context) error {
	detail, err := h.service.Get(adminKey(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	return writeAdminDetail(c, http.StatusOK, detail)
}

func (h *adminHandler) parse(c *echo.Context) error {
	document, err := h.service.ParseDefinition(httpx.Body(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, map[string]any{"json": document})
}

func (h *adminHandler) validate(c *echo.Context) error {
	validation, err := h.service.Validate(httpx.Body(c), strings.ToLower(httpx.Query(c, "key")))
	if err != nil {
		return adminServiceError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, validation)
}

func (h *adminHandler) test(c *echo.Context) error {
	result, err := h.service.Test(httpx.Body(c), strings.ToLower(httpx.Query(c, "key")))
	if err != nil {
		return adminServiceError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, result)
}

func (h *adminHandler) create(c *echo.Context) error {
	detail, err := h.service.Create(httpx.Body(c), h.security.RequestAuthor(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	return writeAdminDetail(c, http.StatusCreated, detail)
}

func (h *adminHandler) update(c *echo.Context) error {
	expectedVersion, err := adminExpectedVersion(c)
	if err != nil {
		return adminServiceError(c, err)
	}
	detail, err := h.service.Update(adminKey(c), httpx.Body(c), expectedVersion, h.security.RequestAuthor(c))
	if err != nil {
		return adminServiceError(c, err)
	}
	// A renamed endpoint must not show up under its old key in the cached statuses anymore
	_ = cache.DeleteKeysByPattern("endpoint-status-*")
	return writeAdminDetail(c, http.StatusOK, detail)
}

func (h *adminHandler) setEnabled(enabled bool) echo.HandlerFunc {
	return func(c *echo.Context) error {
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

func (h *adminHandler) delete(c *echo.Context) error {
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
	return httpx.JSON(c, http.StatusOK, map[string]any{"key": key, "triggeredAlerts": triggeredAlerts})
}

func adminKey(c *echo.Context) string {
	key, err := url.PathUnescape(c.Param("key"))
	if err != nil {
		key = c.Param("key")
	}
	return strings.ToLower(key)
}

// adminExpectedVersion returns the version from the If-Match header, e.g. "3" or W/"3"
func adminExpectedVersion(c *echo.Context) (int64, error) {
	value := strings.TrimSpace(httpx.Header(c, "If-Match"))
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

func writeAdminDetail(c *echo.Context, status int, detail *managedendpoint.Detail) error {
	if detail.Version > 0 {
		httpx.SetHeader(c, "ETag", fmt.Sprintf(`"%d"`, detail.Version))
	}
	return httpx.JSON(c, status, detail)
}

func adminServiceError(c *echo.Context, err error) error {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, managedendpoint.ErrNotFound), errors.Is(err, common.ErrManagedEndpointNotFound):
		status = http.StatusNotFound
	case errors.Is(err, managedendpoint.ErrReadOnly), errors.Is(err, managedendpoint.ErrKeyConflict), errors.Is(err, common.ErrManagedEndpointAlreadyExists),
		errors.Is(err, common.ErrEndpointKeyInUse), errors.Is(err, common.ErrManagedStatusPageVersionMismatch),
		errors.Is(err, managedendpoint.ErrPushTokenInUse):
		// A managed status page changed by another instance during a rename is a conflict of the rename, not of the
		// version of the endpoint
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
		errors.Is(err, managedendpoint.ErrPushNotTestable), errors.Is(err, managedendpoint.ErrTypeChanged):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		logr.Errorf("[api.adminServiceError] %s", err.Error())
	}
	return httpx.JSON(c, status, map[string]any{"error": err.Error()})
}
