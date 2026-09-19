package statuspage

import (
	"errors"
	"testing"

	pageconfig "gatus/v5/internal/config/statuspage"
)

func TestParse(t *testing.T) {
	page, err := Parse([]byte("slug: infra\ntitle: ' Infra '\ngroups: [core]\nendpoints: [Core_API]\nenabled: true\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Slug != "infra" || page.Title != "Infra" || page.Endpoints[0] != "core_api" || page.Enabled == nil || !*page.Enabled {
		t.Errorf("expected a normalized page, got %+v", page)
	}
	if page, err := Parse([]byte(`{"slug": "apps", "title": "Apps", "groups": ["apps"]}`)); err != nil || page.Slug != "apps" || page.Enabled != nil {
		t.Errorf("expected JSON to be accepted without enabled, got page=%+v err=%v", page, err)
	}
}

func TestParseErrors(t *testing.T) {
	scenarios := []struct {
		name        string
		definition  string
		expectedErr error
	}{
		{name: "empty", definition: "  \n", expectedErr: ErrEmptyDefinition},
		{name: "comment-only", definition: "# nothing\n", expectedErr: ErrEmptyDefinition},
		{name: "unknown-field", definition: "slug: infra\ntitle: Infra\ngroups: [core]\nindexable: true\n", expectedErr: ErrInvalidDefinition},
		{name: "multiple-documents", definition: "slug: infra\ntitle: Infra\ngroups: [core]\n---\nslug: apps\n", expectedErr: ErrInvalidDefinition},
		{name: "wrong-type", definition: "slug: infra\ntitle: Infra\ngroups: core\n", expectedErr: ErrInvalidDefinition},
		{name: "invalid-slug", definition: "slug: Infra\ntitle: Infra\ngroups: [core]\n", expectedErr: pageconfig.ErrInvalidSlug},
		{name: "empty-selection", definition: "slug: infra\ntitle: Infra\n", expectedErr: pageconfig.ErrEmptySelection},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if _, err := Parse([]byte(scenario.definition)); !errors.Is(err, scenario.expectedErr) {
				t.Errorf("expected error %v, got %v", scenario.expectedErr, err)
			}
		})
	}
}

func TestParse_ShowCertificateExpiration(t *testing.T) {
	if page, err := Parse([]byte("slug: infra\ntitle: Infra\ngroups: [core]\nshow-certificate-expiration: true\n")); err != nil || !page.ShowCertificateExpiration {
		t.Errorf("expected show-certificate-expiration to be accepted in YAML, got page=%+v err=%v", page, err)
	}
	if page, err := Parse([]byte(`{"slug": "apps", "title": "Apps", "groups": ["apps"], "show-certificate-expiration": true}`)); err != nil || !page.ShowCertificateExpiration {
		t.Errorf("expected show-certificate-expiration to be accepted in JSON, got page=%+v err=%v", page, err)
	}
	if page, err := Parse([]byte("slug: infra\ntitle: Infra\ngroups: [core]\n")); err != nil || page.ShowCertificateExpiration {
		t.Errorf("expected the certificate expiration to be hidden by default, got page=%+v err=%v", page, err)
	}
}
