package api

import (
	"errors"
	"net/http"

	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/statuspage"
	"gatus/v5/storage/store/common"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

// adminStatusPageError maps the errors of the administration of the status pages to their HTTP status. Unexpected
// errors are logged and answered without their text.
func adminStatusPageError(c *fiber.Ctx, err error) error {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, statuspage.ErrStorageNotSupported):
		status = http.StatusNotImplemented
	case errors.Is(err, statuspage.ErrPageNotFound), errors.Is(err, common.ErrManagedStatusPageNotFound):
		status = http.StatusNotFound
	case errors.Is(err, statuspage.ErrReadOnly), errors.Is(err, statuspage.ErrSlugInUse), errors.Is(err, common.ErrManagedStatusPageAlreadyExists):
		status = http.StatusConflict
	case errors.Is(err, common.ErrManagedStatusPageVersionMismatch), errors.Is(err, common.ErrManagedEndpointVersionMismatch):
		status = http.StatusPreconditionFailed
	case errors.Is(err, errAdminVersionRequired):
		status = http.StatusPreconditionRequired
	case errors.Is(err, statuspage.ErrCycleInProgress), errors.Is(err, statuspage.ErrPageUnavailable):
		status = http.StatusServiceUnavailable
	case errors.Is(err, statuspage.ErrEmptyDefinition), errors.Is(err, statuspage.ErrInvalidDefinition),
		errors.Is(err, statuspage.ErrSlugChanged), errors.Is(err, statuspage.ErrExposureQueryRequired),
		errors.Is(err, pageconfig.ErrInvalidSlug), errors.Is(err, pageconfig.ErrReservedSlug),
		errors.Is(err, pageconfig.ErrInvalidTitle), errors.Is(err, pageconfig.ErrDescriptionTooLong),
		errors.Is(err, pageconfig.ErrEmptySelection), errors.Is(err, pageconfig.ErrInvalidGroups),
		errors.Is(err, pageconfig.ErrInvalidEndpoints), errors.Is(err, pageconfig.ErrInvalidFeatured),
		errors.Is(err, pageconfig.ErrInvalidCharts):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		logr.Errorf("[api.adminStatusPageError] %s", err.Error())
		return c.Status(status).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}
