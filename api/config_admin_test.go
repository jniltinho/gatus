package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gatus/v5/config"
	"gatus/v5/config/admin"
	"gatus/v5/security"
	"github.com/gofiber/fiber/v2"
)

func TestConfigHandler_Admin(t *testing.T) {
	scenarios := []struct {
		name               string
		securityConfig     *security.Config
		adminConfig        *admin.Config
		expectedEnabled    bool
		expectedAuthorized bool
	}{
		{name: "admin-disabled", securityConfig: &security.Config{Basic: &security.BasicConfig{Username: "admin"}}},
		{name: "basic-only", securityConfig: &security.Config{Basic: &security.BasicConfig{Username: "admin"}}, adminConfig: &admin.Config{Enabled: true}, expectedEnabled: true, expectedAuthorized: true},
		{name: "oidc-without-session", securityConfig: &security.Config{OIDC: &security.OIDCConfig{IssuerURL: "https://sso.example.com/", RedirectURL: "http://localhost/authorization-code/callback", Scopes: []string{"openid"}}}, adminConfig: &admin.Config{Enabled: true, AllowedSubjects: []string{"ops@example.com"}}, expectedEnabled: true},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			cfg := &config.Config{Security: scenario.securityConfig, Admin: scenario.adminConfig}
			app := fiber.New()
			app.Get("/api/v1/config", ConfigHandler{securityConfig: cfg.Security, config: cfg}.GetConfig)
			if err := cfg.Security.ApplySecurityMiddleware(app.Group("/protected")); err != nil {
				t.Fatalf("failed to apply security middleware: %v", err)
			}
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			var body struct {
				Admin struct {
					Enabled    bool `json:"enabled"`
					Authorized bool `json:"authorized"`
				} `json:"admin"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("invalid response: %v", err)
			}
			if body.Admin.Enabled != scenario.expectedEnabled || body.Admin.Authorized != scenario.expectedAuthorized {
				t.Errorf("expected enabled=%v authorized=%v, got enabled=%v authorized=%v", scenario.expectedEnabled, scenario.expectedAuthorized, body.Admin.Enabled, body.Admin.Authorized)
			}
		})
	}
}
