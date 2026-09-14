package store

import (
	"gatus/v5/storage/store/common"
	"gatus/v5/storage/store/sql"
)

var _ ManagedEndpointStore = (*sql.Store)(nil)

// ManagedEndpointStore persists the endpoints managed through the administration API.
//
// Every write runs in a transaction and accepts an apply function, called after the write and before the commit:
// if apply returns an error, the transaction is rolled back and that error is returned. If the commit itself fails
// after apply succeeded, the error is returned and the caller must undo what apply did. apply must not wait for
// other writes to the store (with SQLite, the transaction holds the only connection).
type ManagedEndpointStore interface {
	// ListManagedEndpoints returns every managed endpoint, ordered by key
	ListManagedEndpoints() ([]*common.ManagedEndpoint, error)

	// GetManagedEndpoint returns the managed endpoint with the given key, or common.ErrManagedEndpointNotFound
	GetManagedEndpoint(key string) (*common.ManagedEndpoint, error)

	// CreateManagedEndpoint persists a new managed endpoint with version 1, or returns
	// common.ErrManagedEndpointAlreadyExists. On success, Version, CreatedAt and UpdatedAt are set.
	CreateManagedEndpoint(managedEndpoint *common.ManagedEndpoint, apply func() error) error

	// UpdateManagedEndpoint replaces the definition of the managed endpoint if its current version is
	// expectedVersion, or returns common.ErrManagedEndpointVersionMismatch. On success, Version, CreatedAt and
	// UpdatedAt are set.
	UpdateManagedEndpoint(managedEndpoint *common.ManagedEndpoint, expectedVersion int64, apply func() error) error

	// DeleteManagedEndpoint deletes the managed endpoint if its current version is expectedVersion. When
	// deleteEndpointData is true, the statuses, results, events, uptimes and triggered alerts of the key are deleted in
	// the same transaction.
	DeleteManagedEndpoint(key string, expectedVersion int64, deleteEndpointData bool, apply func() error) error
}

// GetManagedEndpointStore returns the storage provider as a ManagedEndpointStore, if it supports managed endpoints
func GetManagedEndpointStore() (ManagedEndpointStore, bool) {
	managedEndpointStore, ok := Get().(ManagedEndpointStore)
	return managedEndpointStore, ok
}
