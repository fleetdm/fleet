package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250814123333, Down_20250814123333)
}

func Up_20250814123333(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE batch_activity_host_results ADD CONSTRAINT unique_batch_host_results_execution_hostid UNIQUE (batch_execution_id, host_id)`); err != nil {
		return fmt.Errorf("adding unique index to batch_activity_host_results: %w", err)
	}
	return nil
}

func Down_20250814123333(tx *sql.Tx) error {
	return nil
}
