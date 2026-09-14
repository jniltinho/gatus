package common

import (
	"errors"
	"time"
)

var (
	// ErrManagedEndpointNotFound is returned when no managed endpoint exists with the given key
	ErrManagedEndpointNotFound = errors.New("managed endpoint not found")

	// ErrManagedEndpointAlreadyExists is returned when a managed endpoint with the same key already exists
	ErrManagedEndpointAlreadyExists = errors.New("a managed endpoint with the same key already exists")

	// ErrManagedEndpointVersionMismatch is returned when the managed endpoint was changed since the version provided
	ErrManagedEndpointVersionMismatch = errors.New("managed endpoint version does not match")
)

// ManagedEndpoint is the persisted definition of an endpoint managed through the administration API
type ManagedEndpoint struct {
	// Key is the key of the endpoint (group and name)
	Key string

	// Definition is the endpoint definition in YAML, as submitted and without default values
	Definition string

	// Version starts at 1 and is incremented on every change
	Version int64

	CreatedAt time.Time
	UpdatedAt time.Time

	// UpdatedBy is the author of the last change (basic auth username or OIDC subject)
	UpdatedBy string
}
