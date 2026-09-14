package sql

import (
	"time"

	"gatus/v5/storage/store/common"
)

// GetEndpointSummaries returns the latest maximumResults results and the uptimes of the endpoints with the given keys,
// reading only the columns that can be published, in a single read transaction with three queries
func (s *Store) GetEndpointSummaries(keys []string, maximumResults int, now time.Time) (map[string]*common.EndpointSummary, error) {
	summaries := make(map[string]*common.EndpointSummary, len(keys))
	uniqueKeys := uniqueStrings(keys)
	if len(uniqueKeys) == 0 {
		return summaries, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	// Nothing is written, so the transaction is always rolled back
	defer func() { _ = tx.Rollback() }()
	args, placeholders := appendPlaceholders(nil, uniqueKeys)
	rows, err := tx.Query("SELECT endpoint_id, endpoint_key FROM endpoints WHERE endpoint_key IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	keysByID := make(map[int64]string, len(uniqueKeys))
	for rows.Next() {
		var id int64
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			_ = rows.Close()
			return nil, err
		}
		keysByID[id] = key
		summaries[key] = &common.EndpointSummary{Results: []common.ResultSummary{}}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(keysByID) == 0 {
		return summaries, nil
	}
	if maximumResults > 0 {
		ids := make([]int64, 0, len(keysByID))
		for id := range keysByID {
			ids = append(ids, id)
		}
		args, placeholders := appendPlaceholders([]any{maximumResults}, ids)
		rows, err := tx.Query(`
			SELECT endpoint_id, success, duration, timestamp
			FROM (
				SELECT endpoint_id, endpoint_result_id, success, duration, timestamp,
					ROW_NUMBER() OVER (PARTITION BY endpoint_id ORDER BY endpoint_result_id DESC) AS rn
				FROM endpoint_results
				WHERE endpoint_id IN (`+placeholders+`)
			) recent_results
			WHERE rn <= $1
			ORDER BY endpoint_id, endpoint_result_id
		`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			var result common.ResultSummary
			if err := rows.Scan(&id, &result.Success, &result.Duration, &result.Timestamp); err != nil {
				_ = rows.Close()
				return nil, err
			}
			summary := summaries[keysByID[id]]
			summary.Results = append(summary.Results, result)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	existingKeys := make([]string, 0, len(keysByID))
	for _, key := range keysByID {
		existingKeys = append(existingKeys, key)
	}
	uptimes, err := queryUptimesByKeys(tx, existingKeys, now)
	if err != nil {
		return nil, err
	}
	for key, endpointUptimes := range uptimes {
		summaries[key].Uptimes = *endpointUptimes
	}
	return summaries, nil
}
