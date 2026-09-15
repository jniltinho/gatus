package sql

import (
	"database/sql"
	"strconv"

	"gatus/v5/config/endpoint"
)

// createEndpointResultMessagesSchema creates the table of the messages and origins of the endpoint results (fork). It
// is a table of its own instead of columns of endpoint_results, so that the upstream schema is not altered, and its
// rows are deleted in cascade with their results.
func (s *Store) createEndpointResultMessagesSchema() error {
	switch s.driver {
	case "sqlite":
		_, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS endpoint_result_messages (
				endpoint_result_id INTEGER PRIMARY KEY REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE,
				message            TEXT    NOT NULL,
				origin             TEXT    NOT NULL
			)
		`)
		return err
	case driverMySQL:
		_, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS endpoint_result_messages (
				endpoint_result_id BIGINT      NOT NULL PRIMARY KEY,
				message            TEXT        NOT NULL,
				origin             VARCHAR(16) NOT NULL,
				FOREIGN KEY (endpoint_result_id) REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE
			) ` + mysqlTableOptions)
		return err
	default:
		_, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS endpoint_result_messages (
				endpoint_result_id BIGINT PRIMARY KEY REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE,
				message            TEXT   NOT NULL,
				origin             TEXT   NOT NULL
			)
		`)
		return err
	}
}

// insertEndpointResultMessage stores the message and the origin of a result, if it has any
func (s *Store) insertEndpointResultMessage(tx *sql.Tx, endpointResultID int64, result *endpoint.Result) error {
	if len(result.Message) == 0 && len(result.Origin) == 0 {
		return nil
	}
	_, err := tx.Exec(
		"INSERT INTO endpoint_result_messages (endpoint_result_id, message, origin) VALUES ($1, $2, $3)",
		endpointResultID,
		endpoint.TruncateResultMessage(result.Message),
		result.Origin,
	)
	return err
}

// loadEndpointResultMessages sets the message and the origin of the results in idResultMap that have them
func (s *Store) loadEndpointResultMessages(tx *sql.Tx, idResultMap map[int64]*endpoint.Result) error {
	if len(idResultMap) == 0 {
		return nil
	}
	args := make([]any, 0, len(idResultMap))
	query := "SELECT endpoint_result_id, message, origin FROM endpoint_result_messages WHERE endpoint_result_id IN ("
	index := 1
	for endpointResultID := range idResultMap {
		query += "$" + strconv.Itoa(index) + ","
		args = append(args, endpointResultID)
		index++
	}
	query = query[:len(query)-1] + ")"
	rows, err := tx.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var endpointResultID int64
		var message, origin string
		if err := rows.Scan(&endpointResultID, &message, &origin); err != nil {
			return err
		}
		if result := idResultMap[endpointResultID]; result != nil {
			result.Message, result.Origin = message, origin
		}
	}
	return rows.Err()
}
