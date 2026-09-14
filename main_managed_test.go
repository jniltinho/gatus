package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/managedendpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

// The history of managed endpoints must be preserved on startup and reload, even without admin.enabled
func TestInitializeStorage_PreservesManagedEndpointHistory(t *testing.T) {
	cfg := &config.Config{Storage: &storage.Config{Type: storage.TypeSQLite, Path: filepath.Join(t.TempDir(), "gatus.db"), MaximumNumberOfResults: 100, MaximumNumberOfEvents: 50}}
	if err := store.Initialize(cfg.Storage); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	managed := &endpoint.Endpoint{Name: "site", Group: "web", URL: "https://example.org"}
	orphan := &endpoint.Endpoint{Name: "orphan", Group: "web", URL: "https://example.org"}
	for _, ep := range []*endpoint.Endpoint{managed, orphan} {
		if err := store.Get().InsertEndpointResult(ep, &endpoint.Result{Success: true, Timestamp: time.Now()}); err != nil {
			t.Fatalf("failed to insert result: %v", err)
		}
	}
	managedEndpointStore, _ := store.GetManagedEndpointStore()
	if err := managedEndpointStore.CreateManagedEndpoint(&common.ManagedEndpoint{Key: managed.Key(), Definition: "name: site\ngroup: web\nurl: https://example.org\nconditions: [\"[STATUS] == 200\"]\n"}, nil); err != nil {
		t.Fatalf("failed to create managed endpoint: %v", err)
	}
	store.Get().Close()

	initializeStorage(cfg)
	defer store.Get().Close()

	params := paging.NewEndpointStatusParams().WithResults(1, 20)
	if status, err := store.Get().GetEndpointStatusByKey(managed.Key(), params); err != nil || len(status.Results) != 1 {
		t.Errorf("expected the history of the managed endpoint to be preserved, got status=%v err=%v", status, err)
	}
	if _, err := store.Get().GetEndpointStatusByKey(orphan.Key(), params); err == nil {
		t.Error("expected the history of an endpoint that is neither configured nor managed to be deleted")
	}
	if managedendpoint.EndpointByKey(managed.Key()) == nil {
		t.Error("expected the managed endpoint to be loaded")
	}
}
