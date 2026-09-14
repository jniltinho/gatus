package sql

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"gatus/v5/storage/store/common"
)

const managedEndpointColumns = "endpoint_key, definition, version, created_at, updated_at, updated_by"

// createManagedEndpointsSchema creates the table of the endpoints managed through the administration API.
// Timestamps are stored as Unix milliseconds so that both drivers behave the same way.
func (s *Store) createManagedEndpointsSchema() error {
	if s.driver == "sqlite" {
		_, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS managed_endpoints (
				managed_endpoint_id INTEGER PRIMARY KEY,
				endpoint_key        TEXT    NOT NULL UNIQUE,
				definition          TEXT    NOT NULL,
				version             INTEGER NOT NULL,
				created_at          INTEGER NOT NULL,
				updated_at          INTEGER NOT NULL,
				updated_by          TEXT    NOT NULL
			)
		`)
		return err
	}
	if s.driver == driverMySQL {
		_, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS managed_endpoints (
				managed_endpoint_id BIGINT       AUTO_INCREMENT PRIMARY KEY,
				endpoint_key        VARCHAR(768) NOT NULL UNIQUE,
				definition          MEDIUMTEXT   NOT NULL,
				version             BIGINT       NOT NULL,
				created_at          BIGINT       NOT NULL,
				updated_at          BIGINT       NOT NULL,
				updated_by          MEDIUMTEXT   NOT NULL
			) ` + mysqlTableOptions)
		return err
	}
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS managed_endpoints (
			managed_endpoint_id BIGSERIAL PRIMARY KEY,
			endpoint_key        TEXT      NOT NULL UNIQUE,
			definition          TEXT      NOT NULL,
			version             BIGINT    NOT NULL,
			created_at          BIGINT    NOT NULL,
			updated_at          BIGINT    NOT NULL,
			updated_by          TEXT      NOT NULL
		)
	`)
	return err
}

// ListManagedEndpoints returns every managed endpoint, ordered by key
func (s *Store) ListManagedEndpoints() ([]*common.ManagedEndpoint, error) {
	rows, err := s.db.Query("SELECT " + managedEndpointColumns + " FROM managed_endpoints ORDER BY endpoint_key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var managedEndpoints []*common.ManagedEndpoint
	for rows.Next() {
		managedEndpoint, err := scanManagedEndpoint(rows)
		if err != nil {
			return nil, err
		}
		managedEndpoints = append(managedEndpoints, managedEndpoint)
	}
	return managedEndpoints, rows.Err()
}

// GetManagedEndpoint returns the managed endpoint with the given key, or common.ErrManagedEndpointNotFound
func (s *Store) GetManagedEndpoint(key string) (*common.ManagedEndpoint, error) {
	managedEndpoint, err := scanManagedEndpoint(s.db.QueryRow("SELECT "+managedEndpointColumns+" FROM managed_endpoints WHERE endpoint_key = $1", key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, common.ErrManagedEndpointNotFound
	}
	return managedEndpoint, err
}

// CreateManagedEndpoint persists a new managed endpoint with version 1
func (s *Store) CreateManagedEndpoint(managedEndpoint *common.ManagedEndpoint, apply func() error) error {
	now := time.Now().UnixMilli()
	err := s.inTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			"INSERT INTO managed_endpoints ("+managedEndpointColumns+") VALUES ($1, $2, $3, $4, $5, $6)",
			managedEndpoint.Key, managedEndpoint.Definition, 1, now, now, managedEndpoint.UpdatedBy,
		)
		if isUniqueViolation(err) {
			return common.ErrManagedEndpointAlreadyExists
		}
		return err
	}, apply)
	if err != nil {
		return err
	}
	managedEndpoint.Version = 1
	managedEndpoint.CreatedAt, managedEndpoint.UpdatedAt = time.UnixMilli(now), time.UnixMilli(now)
	return nil
}

// UpdateManagedEndpoint replaces the definition of the managed endpoint if its current version is expectedVersion
func (s *Store) UpdateManagedEndpoint(managedEndpoint *common.ManagedEndpoint, expectedVersion int64, apply func() error) error {
	now := time.Now().UnixMilli()
	var createdAt int64
	err := s.inTransaction(func(tx *sql.Tx) error {
		var version int64
		err := tx.QueryRow("SELECT version, created_at FROM managed_endpoints WHERE endpoint_key = $1", managedEndpoint.Key).Scan(&version, &createdAt)
		if errors.Is(err, sql.ErrNoRows) {
			return common.ErrManagedEndpointNotFound
		}
		if err != nil {
			return err
		}
		if version != expectedVersion {
			return common.ErrManagedEndpointVersionMismatch
		}
		result, err := tx.Exec(
			"UPDATE managed_endpoints SET definition = $1, version = $2, updated_at = $3, updated_by = $4 WHERE endpoint_key = $5 AND version = $6",
			managedEndpoint.Definition, expectedVersion+1, now, managedEndpoint.UpdatedBy, managedEndpoint.Key, expectedVersion,
		)
		if err != nil {
			return err
		}
		// Another transaction may have changed the row between the SELECT and the UPDATE
		if rowsAffected, err := result.RowsAffected(); err != nil || rowsAffected != 1 {
			return common.ErrManagedEndpointVersionMismatch
		}
		return nil
	}, apply)
	if err != nil {
		return err
	}
	managedEndpoint.Version = expectedVersion + 1
	managedEndpoint.CreatedAt, managedEndpoint.UpdatedAt = time.UnixMilli(createdAt), time.UnixMilli(now)
	return nil
}

// DeleteManagedEndpoint deletes the managed endpoint if its current version is expectedVersion and, when
// deleteEndpointData is true, the statuses, results, events, uptimes and triggered alerts of its key
func (s *Store) DeleteManagedEndpoint(key string, expectedVersion int64, deleteEndpointData bool, apply func() error) error {
	err := s.inTransaction(func(tx *sql.Tx) error {
		result, err := tx.Exec("DELETE FROM managed_endpoints WHERE endpoint_key = $1 AND version = $2", key, expectedVersion)
		if err != nil {
			return err
		}
		if rowsAffected, err := result.RowsAffected(); err != nil {
			return err
		} else if rowsAffected == 0 {
			var version int64
			err := tx.QueryRow("SELECT version FROM managed_endpoints WHERE endpoint_key = $1", key).Scan(&version)
			if errors.Is(err, sql.ErrNoRows) {
				return common.ErrManagedEndpointNotFound
			}
			if err != nil {
				return err
			}
			return common.ErrManagedEndpointVersionMismatch
		}
		if deleteEndpointData {
			// Results, events, uptimes and triggered alerts are deleted through ON DELETE CASCADE
			if _, err := tx.Exec("DELETE FROM endpoints WHERE endpoint_key = $1", key); err != nil {
				return err
			}
		}
		return nil
	}, apply)
	if err == nil && deleteEndpointData && s.writeThroughCache != nil {
		_ = s.writeThroughCache.DeleteKeysByPattern(key + "*")
	}
	return err
}

// inTransaction runs write in a transaction, then apply, and only commits if both succeed
func (s *Store) inTransaction(write func(tx *sql.Tx) error, apply func() error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err = write(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if apply != nil {
		if err = apply(); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanManagedEndpoint(row rowScanner) (*common.ManagedEndpoint, error) {
	var managedEndpoint common.ManagedEndpoint
	var createdAt, updatedAt int64
	if err := row.Scan(&managedEndpoint.Key, &managedEndpoint.Definition, &managedEndpoint.Version, &createdAt, &updatedAt, &managedEndpoint.UpdatedBy); err != nil {
		return nil, err
	}
	managedEndpoint.CreatedAt, managedEndpoint.UpdatedAt = time.UnixMilli(createdAt), time.UnixMilli(updatedAt)
	return &managedEndpoint, nil
}

// isUniqueViolation returns whether err is a unique constraint violation, for both SQLite and PostgreSQL
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") || strings.Contains(message, "duplicate key value violates unique constraint")
}
