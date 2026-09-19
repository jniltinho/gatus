package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"gatus/v5/internal/adminbackup"
	"gatus/v5/internal/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/internal/managedendpoint"
	"gatus/v5/internal/security"
	"gatus/v5/internal/statuspage"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

const (
	// adminRestoreMaximumBodySize is the maximum size of the body of the restore routes: an encrypted backup of
	// adminbackup.MaximumPlaintextBytes with the options, below the body limit of 4 MiB of the server
	adminRestoreMaximumBodySize = 3584 * 1024

	adminRestorePreviewPath = "/api/v1/admin/restore/preview"
	adminRestorePath        = "/api/v1/admin/restore"

	// restorePasswordWindow, restorePasswordMaximumFailures and restorePasswordMaximumKeys limit the wrong passwords of
	// the restores per client, independently of the login
	restorePasswordWindow          = 15 * time.Minute
	restorePasswordMaximumFailures = 10
	restorePasswordMaximumKeys     = 10000

	derivationRetryAfterSeconds = "5"
)

// restorePasswordLimiter counts the wrong passwords of the restores per client
var restorePasswordLimiter = security.NewFailureLimiter(restorePasswordWindow, restorePasswordMaximumFailures, restorePasswordMaximumKeys)

func isAdminRestorePath(path string) bool {
	return path == adminRestorePreviewPath || path == adminRestorePath
}

type adminBackupHandler struct {
	security *security.Config
	restorer *adminbackup.Restorer
}

type backupRequest struct {
	Password string `json:"password"`
}

type restoreRequest struct {
	File             json.RawMessage `json:"file"`
	Password         string          `json:"password"`
	Overwrite        bool            `json:"overwrite"`
	DisableEndpoints bool            `json:"disableEndpoints"`
	Fingerprint      string          `json:"fingerprint"`
}

// registerAdminBackupRoutes registers the backup and restore routes of the administration (fork). The router must
// already require authentication, administrator permission and request protection; the restore routes are exempted
// from its body limit and check their own.
func registerAdminBackupRoutes(router httpx.Router, cfg *config.Config) {
	handler := &adminBackupHandler{security: cfg.Security, restorer: &adminbackup.Restorer{Endpoints: managedendpoint.NewService(cfg), StatusPages: statuspage.NewService()}}
	clientIP := clientIPMiddleware(cfg.StatusPages.TrustedProxyPrefixes())
	router.POST("/backup", handler.backup, requireJSON)
	router.POST("/restore/preview", handler.preview, requireJSON, clientIP)
	router.POST("/restore", handler.apply, requireJSON, clientIP)
}

// requireJSON rejects the requests whose content type is not application/json, even without body
func requireJSON(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
		mediaType, _, err := mime.ParseMediaType(httpx.Header(c, echo.HeaderContentType))
		if err != nil || mediaType != echo.MIMEApplicationJSON {
			return adminError(c, http.StatusUnsupportedMediaType, "content type must be application/json")
		}
		return next(c)
	}
}

func (h *adminBackupHandler) backup(c *echo.Context) error {
	var request backupRequest
	if body := bytes.TrimSpace(httpx.Body(c)); len(body) > 0 {
		if err := decodeAdminJSON(body, &request); err != nil {
			return adminError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		}
	}
	backup, err := adminbackup.Build(h.security.RequestAuthor(c), request.Password)
	if err != nil {
		return backupError(c, err)
	}
	httpx.SetHeader(c, echo.HeaderContentDisposition, `attachment; filename="`+backup.Filename+`"`)
	httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
	return httpx.Send(c, http.StatusOK, backup.Body)
}

func (h *adminBackupHandler) preview(c *echo.Context) error {
	request, plaintext, sent, err := h.readRestore(c)
	if sent {
		return err
	}
	plan, err := h.restorer.Plan(plaintext, adminbackup.Options{Overwrite: request.Overwrite, DisableEndpoints: request.DisableEndpoints})
	if err != nil {
		return backupError(c, err)
	}
	return httpx.JSON(c, http.StatusOK, plan)
}

func (h *adminBackupHandler) apply(c *echo.Context) error {
	request, plaintext, sent, err := h.readRestore(c)
	if sent {
		return err
	}
	if len(request.Fingerprint) == 0 {
		return adminError(c, http.StatusBadRequest, "the fingerprint of the preview is required")
	}
	result, err := h.restorer.Apply(plaintext, adminbackup.Options{Overwrite: request.Overwrite, DisableEndpoints: request.DisableEndpoints}, request.Fingerprint, h.security.RequestAuthor(c))
	if err != nil {
		return backupError(c, err)
	}
	// The statuses of the endpoints are cached by the API, like after the changes of api/admin.go
	_ = cache.DeleteKeysByPattern("endpoint-status-*")
	return httpx.JSON(c, http.StatusOK, result)
}

// readRestore reads the body of a restore and returns the plaintext of its backup file. When sent is true, the error
// response is already set and the handler must return err.
func (h *adminBackupHandler) readRestore(c *echo.Context) (request *restoreRequest, plaintext []byte, sent bool, err error) {
	body := httpx.Body(c)
	if len(body) > adminRestoreMaximumBodySize {
		return nil, nil, true, adminError(c, http.StatusRequestEntityTooLarge, "request body is too large")
	}
	request = &restoreRequest{}
	if err := decodeAdminJSON(body, request); err != nil {
		return nil, nil, true, adminError(c, http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if len(bytes.TrimSpace(request.File)) == 0 || bytes.Equal(bytes.TrimSpace(request.File), []byte("null")) {
		return nil, nil, true, adminError(c, http.StatusBadRequest, "the backup file is required")
	}
	clientIP, _ := c.Get(security.LocalsClientIP).(netip.Addr)
	now := time.Now()
	if blocked, retryAfter := restorePasswordLimiter.Blocked(clientIP, now); blocked {
		httpx.SetHeader(c, echo.HeaderRetryAfter, strconv.Itoa(int(retryAfter.Seconds())))
		return nil, nil, true, adminError(c, http.StatusTooManyRequests, "too many wrong passwords, try again later")
	}
	plaintext, _, err = adminbackup.Unwrap(request.File, request.Password)
	if err != nil {
		if errors.Is(err, adminbackup.ErrInvalidPassword) {
			restorePasswordLimiter.Failure(clientIP, now)
			logr.Warnf("[api.readRestore] Wrong password for an encrypted backup from %s", clientIP)
		}
		return nil, nil, true, backupError(c, err)
	}
	return request, plaintext, false, nil
}

// decodeAdminJSON decodes a JSON object, rejecting unknown fields and trailing data
func decodeAdminJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("unexpected data after the JSON object")
	}
	return nil
}

func backupError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, adminbackup.ErrBusy):
		httpx.SetHeader(c, echo.HeaderRetryAfter, derivationRetryAfterSeconds)
		return adminError(c, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, adminbackup.ErrTooLarge):
		return adminError(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, adminbackup.ErrFingerprintMismatch):
		return adminError(c, http.StatusConflict, err.Error())
	case errors.Is(err, adminbackup.ErrCycleInProgress), errors.Is(err, adminbackup.ErrUnavailable):
		return adminError(c, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, adminbackup.ErrStorageNotSupported):
		return adminError(c, http.StatusNotImplemented, err.Error())
	case errors.Is(err, adminbackup.ErrInvalidFile), errors.Is(err, adminbackup.ErrInvalidPassword), errors.Is(err, adminbackup.ErrPasswordRequired),
		errors.Is(err, adminbackup.ErrNotEncrypted), errors.Is(err, adminbackup.ErrPasswordLength):
		return adminError(c, http.StatusBadRequest, err.Error())
	default:
		logr.Errorf("[api.backupError] Backup or restore failed: %s", err.Error())
		return adminError(c, http.StatusInternalServerError, "the backup or the restore failed")
	}
}
