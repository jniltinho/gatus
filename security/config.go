package security

import (
	"encoding/base64"
	"net/http"
	"sync"

	g8 "github.com/TwiN/g8/v2"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

const (
	cookieNameState   = "gatus_state"
	cookieNameNonce   = "gatus_nonce"
	cookieNameSession = "gatus_session"
)

// Config is the security configuration for Gatus
type Config struct {
	Basic *BasicConfig `yaml:"basic,omitempty"`
	OIDC  *OIDCConfig  `yaml:"oidc,omitempty"`

	gate *g8.Gate

	// Limiter of the authentication failures of security.basic (fork, see basic_auth.go)
	limiterOnce sync.Once
	limiter     *failureLimiter
}

// ValidateAndSetDefaults returns whether the security configuration is valid or not and sets default values.
func (c *Config) ValidateAndSetDefaults() bool {
	return (c.Basic == nil || c.Basic.validateAndSetDefaults()) && (c.OIDC == nil || c.OIDC.ValidateAndSetDefaults())
}

// RegisterHandlers registers all handlers required based on the security configuration
func (c *Config) RegisterHandlers(router fiber.Router) error {
	if c.OIDC != nil {
		if err := c.OIDC.initialize(); err != nil {
			return err
		}
		router.All("/oidc/login", c.OIDC.loginHandler)
		router.All("/authorization-code/callback", adaptor.HTTPHandlerFunc(c.OIDC.callbackHandler))
	}
	return nil
}

// ApplySecurityMiddleware applies an authentication middleware to the router passed.
// The router passed should be a sub-router in charge of handlers that require authentication.
func (c *Config) ApplySecurityMiddleware(router fiber.Router) error {
	if c.OIDC != nil {
		// We're going to use g8 for session handling
		clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
			if _, exists := sessions.Get(token); exists {
				return g8.NewClient(token)
			}
			return nil
		})
		customTokenExtractorFunc := func(request *http.Request) string {
			sessionCookie, err := request.Cookie(cookieNameSession)
			if err != nil {
				return ""
			}
			return sessionCookie.Value
		}
		// TODO: g8: Add a way to update cookie after? would need the writer
		authorizationService := g8.NewAuthorizationService().WithClientProvider(clientProvider)
		c.gate = g8.New().WithAuthorizationService(authorizationService).WithCustomTokenExtractor(customTokenExtractorFunc)
		router.Use(adaptor.HTTPMiddleware(c.gate.Protect))
	} else if c.Basic != nil {
		if _, err := base64.URLEncoding.DecodeString(c.Basic.PasswordBcryptHashBase64Encoded); err != nil {
			return err
		}
		// Login sessions of the login screen or Authorization: Basic, under the failure limiter (fork, see basic_auth.go)
		router.Use(c.basicMiddleware)
	}
	return nil
}

// IsAuthenticated checks whether the user is authenticated
// If the Config does not warrant authentication, it will always return true.
func (c *Config) IsAuthenticated(ctx *fiber.Ctx) bool {
	if c.UsesBasicLogin() {
		return c.authenticateBasic(ctx).authenticated
	}
	if c.gate != nil {
		// TODO: Update g8 to support fasthttp natively? (see g8's fasthttp branch)
		request, err := adaptor.ConvertRequest(ctx, false)
		if err != nil {
			logr.Errorf("[security.IsAuthenticated] Unexpected error converting request: %v", err)
			return false
		}
		token := c.gate.ExtractTokenFromRequest(request)
		_, hasSession := sessions.Get(token)
		return hasSession
	}
	return false
}
