package api

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"

	pushconfig "gatus/v5/config/push"
	"gatus/v5/internal/httpx"
	"gatus/v5/pushkey"
	"gatus/v5/security"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

// createPushKeyRequest is the body of POST /api/v1/admin/push-keys
type createPushKeyRequest struct {
	Name string `json:"name"`
}

// registerAdminPushKeyRoutes registers the routes of the administration of the global push keys (fork). The router must
// already require authentication, administrator permission and request protection.
func registerAdminPushKeyRoutes(router httpx.Router, securityConfig *security.Config) {
	httpx.GetAndHead(router, "/push-keys", func(c *echo.Context) error {
		return httpx.JSON(c, http.StatusOK, pushkey.List())
	})
	router.POST("/push-keys", func(c *echo.Context) error {
		var request createPushKeyRequest
		// Only JSON, as before: the body was already read ahead, see httpx.BufferBody
		mediaType, _, _ := mime.ParseMediaType(httpx.Header(c, echo.HeaderContentType))
		if err := json.Unmarshal(httpx.Body(c), &request); err != nil || mediaType != echo.MIMEApplicationJSON {
			return httpx.JSON(c, http.StatusBadRequest, map[string]any{"error": "invalid body: a JSON object with the name of the key is expected"})
		}
		created, err := pushkey.Create(request.Name, securityConfig.RequestAuthor(c))
		if err != nil {
			return adminPushKeyError(c, err)
		}
		// The token is only available in this response
		httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
		return httpx.JSON(c, http.StatusCreated, created)
	})
	router.DELETE("/push-keys/:id", func(c *echo.Context) error {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			return adminPushKeyError(c, pushkey.ErrNotFound)
		}
		if err := pushkey.Delete(id, securityConfig.RequestAuthor(c)); err != nil {
			return adminPushKeyError(c, err)
		}
		return httpx.JSON(c, http.StatusOK, map[string]any{"id": id})
	})
}

// adminPushKeyError maps the errors of the administration of the push keys to their HTTP status. Unexpected errors are
// logged and answered without their text.
func adminPushKeyError(c *echo.Context, err error) error {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, pushkey.ErrStorageNotSupported):
		status = http.StatusNotImplemented
	case errors.Is(err, pushkey.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, pushkey.ErrNameInUse):
		status = http.StatusConflict
	case errors.Is(err, pushkey.ErrCycleInProgress):
		status = http.StatusServiceUnavailable
	case errors.Is(err, pushconfig.ErrInvalidKeyName):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		logr.Errorf("[api.adminPushKeyError] %s", err.Error())
		return httpx.JSON(c, status, map[string]any{"error": "internal error"})
	}
	return httpx.JSON(c, status, map[string]any{"error": err.Error()})
}
