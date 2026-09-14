package sql

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gatus/v5/config/endpoint"
	"gatus/v5/storage"
	"gatus/v5/storage/store/common"
	"gatus/v5/storage/store/common/paging"
)

const managedEndpointTestGroup = "managed-test"

// managedEndpointTestStores returns a SQLite store and, when GATUS_TEST_POSTGRES_URL, GATUS_TEST_MYSQL_URL or
// GATUS_TEST_MARIADB_URL are set, a PostgreSQL, MySQL or MariaDB store. MySQL and MariaDB stores use a database of their
// own, removed when the test ends.
func managedEndpointTestStores(t *testing.T) map[string]*Store {
	t.Helper()
	stores := make(map[string]*Store)
	sqliteStore, err := NewStore("sqlite", filepath.Join(t.TempDir(), "managed.db"), true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	t.Cleanup(sqliteStore.Close)
	stores["sqlite"] = sqliteStore
	if postgresURL := os.Getenv("GATUS_TEST_POSTGRES_URL"); postgresURL == "" {
		t.Log("GATUS_TEST_POSTGRES_URL is not set, skipping PostgreSQL")
	} else {
		postgresStore, err := NewStore("postgres", postgresURL, true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
		if err != nil {
			t.Fatalf("failed to create postgres store: %v", err)
		}
		t.Cleanup(postgresStore.Close)
		for _, query := range []string{"DELETE FROM managed_endpoints", "DELETE FROM endpoints WHERE endpoint_group = '" + managedEndpointTestGroup + "'"} {
			if _, err := postgresStore.db.Exec(query); err != nil {
				t.Fatalf("failed to clean postgres database: %v", err)
			}
		}
		stores["postgres"] = postgresStore
	}
	for name, dsn := range mysqlTestServers() {
		mysqlStore, err := NewStore(driverMySQL, newMySQLTestDatabase(t, dsn), true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
		if err != nil {
			t.Fatalf("failed to create %s store: %v", name, err)
		}
		t.Cleanup(mysqlStore.Close)
		stores[name] = mysqlStore
	}
	return stores
}

func newManagedEndpointTestEndpoint() *endpoint.Endpoint {
	return &endpoint.Endpoint{Name: "api", Group: managedEndpointTestGroup, URL: "https://example.org"}
}

func TestStore_ManagedEndpoints(t *testing.T) {
	for driver, store := range managedEndpointTestStores(t) {
		t.Run(driver, func(t *testing.T) {
			key := newManagedEndpointTestEndpoint().Key()
			created := &common.ManagedEndpoint{Key: key, Definition: "name: api\n", UpdatedBy: "ops@example.com"}
			if err := store.CreateManagedEndpoint(created, nil); err != nil {
				t.Fatalf("failed to create managed endpoint: %v", err)
			}
			if created.Version != 1 || created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
				t.Errorf("expected version 1 and matching timestamps, got version=%d createdAt=%s updatedAt=%s", created.Version, created.CreatedAt, created.UpdatedAt)
			}
			if err := store.CreateManagedEndpoint(&common.ManagedEndpoint{Key: key, Definition: "name: api\n"}, nil); !errors.Is(err, common.ErrManagedEndpointAlreadyExists) {
				t.Errorf("expected ErrManagedEndpointAlreadyExists, got %v", err)
			}
			fetched, err := store.GetManagedEndpoint(key)
			if err != nil {
				t.Fatalf("failed to get managed endpoint: %v", err)
			}
			if fetched.Definition != created.Definition || fetched.Version != 1 || fetched.UpdatedBy != "ops@example.com" || !fetched.CreatedAt.Equal(created.CreatedAt) {
				t.Errorf("unexpected managed endpoint: %+v", fetched)
			}
			if list, err := store.ListManagedEndpoints(); err != nil || len(list) != 1 || list[0].Key != key {
				t.Errorf("expected a list with the managed endpoint, got %v (err=%v)", list, err)
			}

			time.Sleep(2 * time.Millisecond)
			updated := &common.ManagedEndpoint{Key: key, Definition: "name: api\ninterval: 5m\n", UpdatedBy: "dev@example.com"}
			if err := store.UpdateManagedEndpoint(updated, 2, nil); !errors.Is(err, common.ErrManagedEndpointVersionMismatch) {
				t.Errorf("expected ErrManagedEndpointVersionMismatch, got %v", err)
			}
			if err := store.UpdateManagedEndpoint(updated, 1, nil); err != nil {
				t.Fatalf("failed to update managed endpoint: %v", err)
			}
			if updated.Version != 2 || !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.After(created.UpdatedAt) {
				t.Errorf("expected version 2, same creation time and a later update time, got %+v", updated)
			}
			if fetched, err := store.GetManagedEndpoint(key); err != nil || fetched.Definition != updated.Definition || fetched.Version != 2 || fetched.UpdatedBy != "dev@example.com" {
				t.Errorf("expected the updated managed endpoint, got %+v (err=%v)", fetched, err)
			}
			if err := store.UpdateManagedEndpoint(&common.ManagedEndpoint{Key: "unknown", Definition: "name: x\n"}, 1, nil); !errors.Is(err, common.ErrManagedEndpointNotFound) {
				t.Errorf("expected ErrManagedEndpointNotFound, got %v", err)
			}

			if err := store.DeleteManagedEndpoint(key, 1, false, nil); !errors.Is(err, common.ErrManagedEndpointVersionMismatch) {
				t.Errorf("expected ErrManagedEndpointVersionMismatch, got %v", err)
			}
			if err := store.DeleteManagedEndpoint(key, 2, false, nil); err != nil {
				t.Fatalf("failed to delete managed endpoint: %v", err)
			}
			if _, err := store.GetManagedEndpoint(key); !errors.Is(err, common.ErrManagedEndpointNotFound) {
				t.Errorf("expected ErrManagedEndpointNotFound after deletion, got %v", err)
			}
			if err := store.DeleteManagedEndpoint(key, 2, false, nil); !errors.Is(err, common.ErrManagedEndpointNotFound) {
				t.Errorf("expected ErrManagedEndpointNotFound when deleting twice, got %v", err)
			}
		})
	}
}

func TestStore_ManagedEndpoints_ApplyErrorRollsBack(t *testing.T) {
	errApply := errors.New("failed to apply")
	failingApply := func() error { return errApply }
	for driver, store := range managedEndpointTestStores(t) {
		t.Run(driver, func(t *testing.T) {
			key := newManagedEndpointTestEndpoint().Key()
			if err := store.CreateManagedEndpoint(&common.ManagedEndpoint{Key: key, Definition: "name: api\n"}, failingApply); !errors.Is(err, errApply) {
				t.Fatalf("expected the apply error, got %v", err)
			}
			if _, err := store.GetManagedEndpoint(key); !errors.Is(err, common.ErrManagedEndpointNotFound) {
				t.Fatalf("expected the creation to be rolled back, got %v", err)
			}
			applied := false
			if err := store.CreateManagedEndpoint(&common.ManagedEndpoint{Key: key, Definition: "name: api\n"}, func() error { applied = true; return nil }); err != nil || !applied {
				t.Fatalf("expected the creation to be applied, got err=%v applied=%v", err, applied)
			}
			if err := store.UpdateManagedEndpoint(&common.ManagedEndpoint{Key: key, Definition: "name: changed\n"}, 1, failingApply); !errors.Is(err, errApply) {
				t.Fatalf("expected the apply error, got %v", err)
			}
			if fetched, err := store.GetManagedEndpoint(key); err != nil || fetched.Definition != "name: api\n" || fetched.Version != 1 {
				t.Errorf("expected the update to be rolled back, got %+v (err=%v)", fetched, err)
			}
			if err := store.DeleteManagedEndpoint(key, 1, true, failingApply); !errors.Is(err, errApply) {
				t.Fatalf("expected the apply error, got %v", err)
			}
			if _, err := store.GetManagedEndpoint(key); err != nil {
				t.Errorf("expected the deletion to be rolled back, got %v", err)
			}
		})
	}
}

func TestStore_DeleteManagedEndpoint_EndpointData(t *testing.T) {
	for driver, store := range managedEndpointTestStores(t) {
		t.Run(driver, func(t *testing.T) {
			ep := newManagedEndpointTestEndpoint()
			if err := store.InsertEndpointResult(ep, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: time.Millisecond}); err != nil {
				t.Fatalf("failed to insert result: %v", err)
			}
			params := paging.NewEndpointStatusParams().WithResults(1, 20)
			if status, err := store.GetEndpointStatusByKey(ep.Key(), params); err != nil || len(status.Results) != 1 {
				t.Fatalf("expected 1 result before deletion, got status=%v err=%v", status, err)
			}
			// Deleting only the definition, as for a managed endpoint in conflict with the configuration file
			if err := store.CreateManagedEndpoint(&common.ManagedEndpoint{Key: ep.Key(), Definition: "name: api\n"}, nil); err != nil {
				t.Fatalf("failed to create managed endpoint: %v", err)
			}
			if err := store.DeleteManagedEndpoint(ep.Key(), 1, false, nil); err != nil {
				t.Fatalf("failed to delete managed endpoint: %v", err)
			}
			if status, err := store.GetEndpointStatusByKey(ep.Key(), params); err != nil || len(status.Results) != 1 {
				t.Errorf("expected the endpoint data to be kept, got status=%v err=%v", status, err)
			}
			// Deleting the definition and the endpoint data
			if err := store.CreateManagedEndpoint(&common.ManagedEndpoint{Key: ep.Key(), Definition: "name: api\n"}, nil); err != nil {
				t.Fatalf("failed to create managed endpoint: %v", err)
			}
			if err := store.DeleteManagedEndpoint(ep.Key(), 1, true, nil); err != nil {
				t.Fatalf("failed to delete managed endpoint: %v", err)
			}
			if _, err := store.GetEndpointStatusByKey(ep.Key(), params); !errors.Is(err, common.ErrEndpointNotFound) {
				t.Errorf("expected ErrEndpointNotFound once the endpoint data is deleted, got %v", err)
			}
		})
	}
}

func TestNewStore_ManagedEndpointsSchemaIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idempotent.db")
	for i := 0; i < 2; i++ {
		store, err := NewStore("sqlite", path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
		if err != nil {
			t.Fatalf("failed to open the store (attempt %d): %v", i+1, err)
		}
		store.Close()
	}
}
