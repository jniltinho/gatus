package store

import (
	"time"

	"gatus/v5/internal/storage/store/common"
	"gatus/v5/internal/storage/store/memory"
	"gatus/v5/internal/storage/store/sql"
)

var (
	_ EndpointUptimeBatchReader = (*memory.Store)(nil)
	_ EndpointUptimeBatchReader = (*sql.Store)(nil)
)

// EndpointUptimeBatchReader reads the uptimes of many endpoints at once, for the public status pages
type EndpointUptimeBatchReader interface {
	// GetUptimesByKeys returns the uptimes over the last 24 hours, 7 days and 30 days before now of the endpoints with
	// the given keys, computed like GetUptimeByKey. Keys without any execution in the last 30 days are absent from the map.
	GetUptimesByKeys(keys []string, now time.Time) (map[string]*common.EndpointUptimes, error)
}

// GetEndpointUptimeBatchReader returns the storage provider as an EndpointUptimeBatchReader, if it supports it
func GetEndpointUptimeBatchReader() (EndpointUptimeBatchReader, bool) {
	reader, ok := Get().(EndpointUptimeBatchReader)
	return reader, ok
}
