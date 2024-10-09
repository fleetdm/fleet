package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240814135330, Down_20240814135330)
}

func Up_20240814135330(tx *sql.Tx) error {
	// This index optimizes finding the most recent query result for a given query and host
	if _, err := tx.Exec(`ALTER TABLE query_results ADD INDEX idx_query_id_host_id_last_fetched (query_id, host_id, last_fetched)`); err != nil {
		return fmt.Errorf("creating query_results index: %w", err)
	}
	return nil
}

func Down_20240814135330(_ *sql.Tx) error {
	return nil
}
