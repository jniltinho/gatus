package statuspage

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gatus/v5/config/endpoint"
	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/storage"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
)

func TestSelect_Featured(t *testing.T) {
	refs := []EndpointRef{
		{Key: "core_web", Name: "web", Group: "core"},
		{Key: "core_api", Name: "api", Group: "core"},
		{Key: "database_pg", Name: "pg", Group: "database"},
	}
	page := &pageconfig.Page{Slug: "infra", Title: "Infra", Groups: []string{"core"}, Featured: []string{"core_api", "missing_endpoint", "database_pg"}}
	selection := Select(page, refs)
	if len(selection.Featured) != 2 || selection.Featured[0].Name != "api" || selection.Featured[1].Name != "pg" {
		t.Fatalf("expected api and pg featured in the order of the page, got %+v", selection.Featured)
	}
	if len(selection.Sections) != 1 || selection.Sections[0].Group != "core" || len(selection.Sections[0].Endpoints) != 1 || selection.Sections[0].Endpoints[0].Name != "web" {
		t.Errorf("expected the featured endpoints to be left out of their section, got %+v", selection.Sections)
	}
	if strings.Join(selection.Keys(), ",") != "core_api,database_pg,core_web" {
		t.Errorf("expected the featured keys first, got %v", selection.Keys())
	}
	if refs := selection.Refs(); len(refs) != 3 || refs[0].Key != "core_api" || refs[2].Key != "core_web" {
		t.Errorf("expected the refs in display order, got %+v", refs)
	}
}

func TestBuildPayload_FeaturedChartsAndResponseTimes(t *testing.T) {
	now := time.Now()
	page := &pageconfig.Page{Slug: "infra", Title: "Infra", Groups: []string{"core"}, Featured: []string{"core_api"}, Charts: []string{"core_api"}}
	selection := Selection{
		Featured: []EndpointRef{{Key: "core_api", Name: "api", Group: "core"}},
		Sections: []Section{{Group: "core", Endpoints: []EndpointRef{{Key: "core_web", Name: "web", Group: "core"}}}},
	}
	average := 120
	summaries := map[string]*common.EndpointSummary{
		"core_api": {Results: []common.ResultSummary{{Timestamp: now, Success: false, Duration: 120 * time.Millisecond}}, Uptimes: common.EndpointUptimes{AverageResponseTime24Hours: &average}},
		"core_web": {Results: []common.ResultSummary{{Timestamp: now, Success: true}}},
	}
	payload := BuildPayload(page, selection, summaries, now)
	if len(payload.Featured) != 1 || payload.Featured[0].Name != "api" || payload.Featured[0].Group != "core" || !payload.Featured[0].Chart {
		t.Fatalf("expected api featured with its group and a chart, got %+v", payload.Featured)
	}
	if *payload.Featured[0].ResponseTime.Last24Hours != 120 || payload.Featured[0].ResponseTime.Last7Days != nil {
		t.Errorf("expected the average response times of the summary, got %+v", payload.Featured[0].ResponseTime)
	}
	if payload.Groups[0].Endpoints[0].Chart {
		t.Error("expected web not to have a chart")
	}
	if payload.Status != StatusDegraded {
		t.Errorf("expected the page status to include the featured endpoints, got %s", payload.Status)
	}
	body, _ := json.Marshal(payload)
	if !strings.Contains(string(body), `"featured":[{"name":"api"`) || !strings.Contains(string(body), `"group":"core"`) || strings.Contains(string(body), "core_api") {
		t.Errorf("unexpected JSON of the featured endpoint: %s", body)
	}
}

func TestSelectionWarnings_FeaturedAndCharts(t *testing.T) {
	refs := []EndpointRef{{Key: "core_web", Name: "web", Group: "core"}, {Key: "database_pg", Name: "pg", Group: "database"}}
	page := &pageconfig.Page{Slug: "infra", Title: "Infra", Groups: []string{"core"}, Featured: []string{"missing_endpoint"}, Charts: []string{"core_web", "database_pg"}}
	warnings := selectionWarnings(page, refs)
	var found []string
	for _, warning := range warnings {
		found = append(found, warning.Type+"="+warning.Value)
	}
	if strings.Join(found, ",") != "featured=missing_endpoint,chart=database_pg" {
		t.Errorf("expected the missing featured endpoint and the chart outside of the page, got %v", found)
	}
}

// countingResponseTimeReader counts the reads of hourly average response times and can fail
type countingResponseTimeReader struct {
	calls atomic.Int32
	err   error
}

func (reader *countingResponseTimeReader) GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error) {
	reader.calls.Add(1)
	if reader.err != nil {
		return nil, reader.err
	}
	return store.Get().GetHourlyAverageResponseTimeByKey(key, from, to)
}

const chartsTestConfig = `
endpoints:
  - name: api
    group: core
    url: https://example.org
    conditions: ["[STATUS] == 200"]
  - name: web
    group: core
    url: https://example.org
    conditions: ["[STATUS] == 200"]
status-pages:
  pages:
    - slug: infra
      title: Infra
      groups: [core]
      featured: [core_api]
      charts: [core_api, core_web]
    - slug: plain
      title: Plain
      groups: [core]
`

func setupResponseTimesTest(t *testing.T, reader *countingResponseTimeReader) {
	t.Helper()
	if err := store.Initialize(&storage.Config{Type: storage.TypeMemory, MaximumNumberOfResults: 100, MaximumNumberOfEvents: 50}); err != nil {
		t.Fatal(err)
	}
	cfg := loadTestConfig(t, chartsTestConfig)
	now := time.Now()
	for _, insert := range []struct {
		ep       *endpoint.Endpoint
		age      time.Duration
		duration time.Duration
	}{
		{cfg.Endpoints[0], 2 * time.Hour, 100 * time.Millisecond},
		{cfg.Endpoints[0], 30 * time.Minute, 300 * time.Millisecond},
		{cfg.Endpoints[1], 10 * time.Minute, 50 * time.Millisecond},
	} {
		if err := store.Get().InsertEndpointResult(insert.ep, &endpoint.Result{Success: true, Timestamp: now.Add(-insert.age), Duration: insert.duration, Hostname: "10.0.0.5"}); err != nil {
			t.Fatal(err)
		}
	}
	Load(cfg)
	previousReader := getResponseTimeReader
	getResponseTimeReader = func() (responseTimeReader, bool) { return reader, true }
	t.Cleanup(func() {
		getResponseTimeReader = previousReader
		publicCache.Clear()
	})
}

func TestPublicResponseTimes(t *testing.T) {
	reader := &countingResponseTimeReader{}
	setupResponseTimesTest(t, reader)
	body, err := PublicResponseTimes("infra", "24h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload ResponseTimesPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Duration != "24h" || len(payload.Endpoints) != 2 || payload.Endpoints[0].Name != "api" || payload.Endpoints[1].Name != "web" {
		t.Fatalf("expected the series of api (featured) then web, got %s", body)
	}
	if api := payload.Endpoints[0]; api.Group != "core" || len(api.Points) != 2 || !api.Points[0].Timestamp.Before(api.Points[1].Timestamp) || api.Points[0].Milliseconds != 100 {
		t.Errorf("expected two hourly points of api in chronological order, got %+v", api)
	}
	if strings.Contains(string(body), "core_api") || strings.Contains(string(body), "10.0.0.5") {
		t.Errorf("expected no key nor hostname in the response times, got %s", body)
	}
	calls := reader.calls.Load()
	if _, err := PublicResponseTimes("infra", "24h"); err != nil || reader.calls.Load() != calls {
		t.Errorf("expected the second request to be served from the cache, got %d reads (err=%v)", reader.calls.Load(), err)
	}
	if body, err := PublicResponseTimes("infra", "7d"); err != nil || !strings.Contains(string(body), `"duration":"7d"`) {
		t.Errorf("expected the 7d response times, got %s (err=%v)", body, err)
	}
	for _, scenario := range []struct{ slug, duration string }{{"infra", "1y"}, {"infra", ""}, {"missing", "24h"}} {
		if _, err := PublicResponseTimes(scenario.slug, scenario.duration); !errors.Is(err, ErrPageNotFound) {
			t.Errorf("expected ErrPageNotFound for %+v, got %v", scenario, err)
		}
	}
	calls = reader.calls.Load()
	if body, err := PublicResponseTimes("plain", "24h"); err != nil || !strings.Contains(string(body), `"endpoints":[]`) || reader.calls.Load() != calls {
		t.Errorf("expected no series and no read for a page without charts, got %s (reads %d → %d, err=%v)", body, calls, reader.calls.Load(), err)
	}
}

func TestPublicResponseTimes_Unavailable(t *testing.T) {
	reader := &countingResponseTimeReader{err: errors.New("database is locked")}
	setupResponseTimesTest(t, reader)
	for i := 0; i < 10; i++ {
		if _, err := PublicResponseTimes("infra", "30d"); !errors.Is(err, ErrPageUnavailable) {
			t.Fatalf("expected ErrPageUnavailable, got %v", err)
		}
	}
	if reader.calls.Load() != 1 {
		t.Errorf("expected a failing storage to be read once, got %d reads", reader.calls.Load())
	}
}
