package api

import (
	"errors"
	"net/http"
	"strconv"

	pushconfig "gatus/v5/config/push"
	"gatus/v5/pushkey"
	"gatus/v5/security"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

// createPushKeyRequest is the body of POST /api/v1/admin/push-keys
type createPushKeyRequest struct {
	Name string `json:"name"`
}

// registerAdminPushKeyRoutes registers the routes of the administration of the global push keys (fork). The router must
// already require authentication, administrator permission and request protection.
func registerAdminPushKeyRoutes(router fiber.Router, securityConfig *security.Config) {
	router.Get("/push-keys", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(pushkey.List())
	})
	router.Post("/push-keys", func(c *fiber.Ctx) error {
		var request createPushKeyRequest
		if err := c.BodyParser(&request); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid body: a JSON object with the name of the key is expected"})
		}
		created, err := pushkey.Create(request.Name, securityConfig.RequestAuthor(c))
		if err != nil {
			return adminPushKeyError(c, err)
		}
		// The token is only available in this response
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Status(http.StatusCreated).JSON(created)
	})
	router.Delete("/push-keys/:id", func(c *fiber.Ctx) error {
		id, err := strconv.ParseInt(c.Params("id"), 10, 64)
		if err != nil || id <= 0 {
			return adminPushKeyError(c, pushkey.ErrNotFound)
		}
		if err := pushkey.Delete(id, securityConfig.RequestAuthor(c)); err != nil {
			return adminPushKeyError(c, err)
		}
		return c.Status(http.StatusOK).JSON(fiber.Map{"id": id})
	})
}

// adminPushKeyError maps the errors of the administration of the push keys to their HTTP status. Unexpected errors are
// logged and answered without their text.
func adminPushKeyError(c *fiber.Ctx, err error) error {
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
		return c.Status(status).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}
