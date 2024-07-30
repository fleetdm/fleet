package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240209110212, Down_20240209110212)
}

func Up_20240209110212(tx *sql.Tx) error {
	stmt := `
		UPDATE scheduled_query_stats
		SET wall_time = wall_time * 1000
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("updating scheduled_query_stats.wall_time: %w", err)
	}

	return nil
}

func Down_20240209110212(tx *sql.Tx) error {
	return nil
}
