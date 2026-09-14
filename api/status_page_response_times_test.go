package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/statuspage"
)

func TestStatusPage_FeaturedAndResponseTimes(t *testing.T) {
	enabled, rateLimit := true, 0
	statusPages := &pageconfig.Config{Enabled: &enabled, RateLimit: &rateLimit, Pages: []*pageconfig.Page{
		{Slug: "infra", Title: "Infra", Groups: []string{"core"}, Featured: []string{"core_api"}, Charts: []string{"core_api"}},
	}}
	app := newStatusPageTestApp(t, statusPageBasicSecurity(t), statusPages)

	response, body := doStatusPageRequest(t, app, http.MethodGet, "/api/v1/status-pages/infra")
	var payload statuspage.Payload
	if response.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &payload) != nil {
		t.Fatalf("expected the page payload, got %d: %s", response.StatusCode, body)
	}
	if len(payload.Featured) != 1 || payload.Featured[0].Name != "api" || !payload.Featured[0].Chart || len(payload.Groups) != 0 {
		t.Errorf("expected api featured with a chart and no other section, got %s", body)
	}

	response, body = doStatusPageRequest(t, app, http.MethodGet, "/api/v1/status-pages/infra/response-times/24h")
	if response.StatusCode != http.StatusOK || response.Header.Get("WWW-Authenticate") != "" || response.Header.Get("Cache-Control") != "no-cache" || response.Header.Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Fatalf("expected the response times without authentication, got %d %v: %s", response.StatusCode, response.Header, body)
	}
	var responseTimes statuspage.ResponseTimesPayload
	if err := json.Unmarshal([]byte(body), &responseTimes); err != nil || len(responseTimes.Endpoints) != 1 || responseTimes.Endpoints[0].Name != "api" || len(responseTimes.Endpoints[0].Points) != 1 {
		t.Errorf("expected one point for api, got %s (err=%v)", body, err)
	}
	for _, sensitive := range []string{"core_api", "10.0.0.5", "secret-error", "example.org"} {
		if strings.Contains(body, sensitive) {
			t.Errorf("expected %q not to be published, got %s", sensitive, body)
		}
	}

	reference, referenceBody := doStatusPageRequest(t, app, http.MethodGet, "/api/v1/status-pages/missing")
	for _, target := range []string{"/api/v1/status-pages/infra/response-times/1y", "/api/v1/status-pages/missing/response-times/24h", "/api/v1/status-pages/infra/response-times/24h/extra"} {
		response, body := doStatusPageRequest(t, app, http.MethodGet, target)
		if response.StatusCode != http.StatusNotFound || body != referenceBody || response.Header.Get("WWW-Authenticate") != "" {
			t.Errorf("%s: expected the identical 404, got %d %s", target, response.StatusCode, body)
		}
		for _, name := range comparedStatusPageHeaders {
			if response.Header.Get(name) != reference.Header.Get(name) {
				t.Errorf("%s: header %s: expected %q, got %q", target, name, reference.Header.Get(name), response.Header.Get(name))
			}
		}
	}
}
