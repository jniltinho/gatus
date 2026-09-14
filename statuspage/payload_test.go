package statuspage

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/storage/store/common"
)

func uptimePointer(value float64) *float64 {
	return &value
}

func TestBuildPayload(t *testing.T) {
	now := time.Date(2026, 9, 14, 15, 0, 0, 0, time.FixedZone("BRT", -3*3600))
	page := &pageconfig.Page{Slug: "infra", Title: "Infraestrutura", Description: "Serviços", Groups: []string{"core", "database", "jobs"}}
	selection := Selection{Sections: []Section{
		{Group: "core", Endpoints: []EndpointRef{{Key: "core_api", Name: "api"}, {Key: "core_web", Name: "web"}, {Key: "core_new", Name: "new"}}},
		{Group: "database", Endpoints: []EndpointRef{{Key: "database_pg", Name: "pg"}}},
		{Group: "jobs", Endpoints: []EndpointRef{{Key: "jobs_batch", Name: "batch"}}},
		{Group: "", Endpoints: []EndpointRef{{Key: "_cdn", Name: "cdn"}}},
	}}
	summaries := map[string]*common.EndpointSummary{
		"core_api": {
			Results: []common.ResultSummary{{Timestamp: now.Add(-2 * time.Minute), Success: false, Duration: 1500 * time.Microsecond}, {Timestamp: now.Add(-time.Minute), Success: true, Duration: 123 * time.Millisecond}},
			Uptimes: common.EndpointUptimes{Last24Hours: uptimePointer(0.5), Last7Days: uptimePointer(0.75)},
		},
		"core_web":    {Results: []common.ResultSummary{{Timestamp: now.Add(-time.Minute), Success: false, Duration: time.Second}}},
		"database_pg": {Results: []common.ResultSummary{{Timestamp: now.Add(-time.Minute), Success: false}}},
		"jobs_batch":  {Results: []common.ResultSummary{}},
	}
	payload := BuildPayload(page, selection, summaries, now)
	if payload.Slug != "infra" || payload.Title != "Infraestrutura" || payload.Description != "Serviços" || !payload.UpdatedAt.Equal(now) || payload.UpdatedAt.Location() != time.UTC {
		t.Errorf("unexpected page fields: %+v", payload)
	}
	if payload.Status != StatusDegraded {
		t.Errorf("expected the page to be degraded, got %s", payload.Status)
	}
	expectedGroupStatuses := []string{StatusDegraded, StatusDown, StatusUnknown, StatusUnknown}
	for i, group := range payload.Groups {
		if group.Status != expectedGroupStatuses[i] {
			t.Errorf("group %q: expected %s, got %s", group.Name, expectedGroupStatuses[i], group.Status)
		}
	}
	api := payload.Groups[0].Endpoints[0]
	if api.Status != StatusUp || len(api.Results) != 2 || api.Results[0].DurationMs != 1 || api.Results[1].DurationMs != 123 || *api.Uptime.Last24Hours != 0.5 || api.Uptime.Last30Days != nil {
		t.Errorf("unexpected api endpoint: %+v", api)
	}
	if newEndpoint := payload.Groups[0].Endpoints[2]; newEndpoint.Status != StatusUnknown || newEndpoint.Results == nil || len(newEndpoint.Results) != 0 {
		t.Errorf("expected an endpoint without summary to be unknown with empty results, got %+v", newEndpoint)
	}
	if payload.Groups[3].Name != "" || payload.Groups[3].Endpoints[0].Name != "cdn" {
		t.Errorf("expected the section without group to keep an empty name, got %+v", payload.Groups[3])
	}
	if BuildPayload(page, Selection{}, nil, now).Status != StatusUnknown {
		t.Error("expected a page without endpoint to be unknown")
	}
}

// Mirror types of the public JSON: decoding with DisallowUnknownFields fails if any other field is published
type allowedPayload struct {
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	UpdatedAt   string         `json:"updatedAt"`
	Truncated   bool           `json:"truncated"`
	Groups      []allowedGroup `json:"groups"`
}

type allowedGroup struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Endpoints []allowedEndpoint `json:"endpoints"`
}

type allowedEndpoint struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Uptime struct {
		Last24Hours *float64 `json:"24h"`
		Last7Days   *float64 `json:"7d"`
		Last30Days  *float64 `json:"30d"`
	} `json:"uptime"`
	Results []struct {
		Timestamp  string `json:"timestamp"`
		Success    bool   `json:"success"`
		DurationMs int64  `json:"durationMs"`
	} `json:"results"`
}

func TestBuildPayload_Allowlist(t *testing.T) {
	page := &pageconfig.Page{Slug: "infra", Title: "Infra", Groups: []string{"core"}}
	selection := Selection{Sections: []Section{{Group: "core", Endpoints: []EndpointRef{{Key: "core_db-master-10-0-0-5", Name: "db"}}}}}
	summaries := map[string]*common.EndpointSummary{"core_db-master-10-0-0-5": {Results: []common.ResultSummary{{Timestamp: time.Now(), Success: true, Duration: time.Millisecond}}}}
	body, err := json.Marshal(BuildPayload(page, selection, summaries, time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var decoded allowedPayload
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("expected only allowed fields, got %v in %s", err, body)
	}
	if strings.Contains(string(body), "10-0-0-5") {
		t.Errorf("expected the endpoint key not to be published, got %s", body)
	}
}
