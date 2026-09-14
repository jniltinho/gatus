package security

import (
	"github.com/TwiN/gatus/v5/config/admin"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// RequestAuthor returns the identity of the authenticated user of the request: the OIDC subject of its session or,
// with basic authentication, the username. It returns an empty string when the request is not authenticated.
func (c *Config) RequestAuthor(ctx *fiber.Ctx) string {
	if c == nil {
		return ""
	}
	if c.OIDC != nil {
		subject, _ := c.sessionSubject(ctx)
		return subject
	}
	if username, ok := ctx.Locals("username").(string); ok {
		return username
	}
	return ""
}

// IsAdmin returns whether the request is authorized to use the administration of endpoints.
//
// With OIDC, the subject of the session must be in admin.allowed-subjects. With basic authentication only, the single
// basic user is an administrator, so this returns true whenever the administration is enabled: the authentication
// itself is enforced by ApplySecurityMiddleware on the protected routes.
func (c *Config) IsAdmin(ctx *fiber.Ctx, adminConfig *admin.Config) bool {
	if c == nil || !adminConfig.IsEnabled() {
		return false
	}
	if c.OIDC != nil {
		subject, ok := c.sessionSubject(ctx)
		return ok && adminConfig.IsSubjectAllowed(subject)
	}
	return c.Basic != nil
}

// AdminMiddleware returns a middleware responding with 403 to requests that are not authorized to use the
// administration. It must be registered after ApplySecurityMiddleware, which responds with 401 to unauthenticated
// requests.
func (c *Config) AdminMiddleware(adminConfig *admin.Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !c.IsAdmin(ctx, adminConfig) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "administrator permission required"})
		}
		return ctx.Next()
	}
}

// sessionSubject returns the OIDC subject of the session of the request, if there is a valid session
func (c *Config) sessionSubject(ctx *fiber.Ctx) (string, bool) {
	if c.gate == nil {
		return "", false
	}
	request, err := adaptor.ConvertRequest(ctx, false)
	if err != nil {
		return "", false
	}
	value, exists := sessions.Get(c.gate.ExtractTokenFromRequest(request))
	if !exists {
		return "", false
	}
	subject, ok := value.(string)
	return subject, ok
}
