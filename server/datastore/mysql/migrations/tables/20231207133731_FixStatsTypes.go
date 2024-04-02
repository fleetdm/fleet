package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231207133731, Down_20231207133731)
}

func Up_20231207133731(tx *sql.Tx) error {
	// Updating some bad data from osquery (which will be ignored in a later PR).
	// Seems safer to update rather than delete.
	stmt := `
		UPDATE scheduled_query_stats SET last_executed = '1970-01-01 00:00:01' WHERE YEAR(last_executed) = '0000';
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("fixing last_executed in scheduled_query_stats: %w", err)
	}

	stmt = `
		ALTER TABLE scheduled_query_stats
		MODIFY COLUMN average_memory BIGINT UNSIGNED NOT NULL,
		MODIFY COLUMN executions BIGINT UNSIGNED NOT NULL,
		MODIFY COLUMN output_size BIGINT UNSIGNED NOT NULL,
		MODIFY COLUMN system_time BIGINT UNSIGNED NOT NULL,
		MODIFY COLUMN user_time BIGINT UNSIGNED NOT NULL,
		MODIFY COLUMN wall_time BIGINT UNSIGNED NOT NULL;
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("changing data types for scheduled_query_stats: %w", err)
	}

	return nil
}

func Down_20231207133731(tx *sql.Tx) error {
	return nil
}
