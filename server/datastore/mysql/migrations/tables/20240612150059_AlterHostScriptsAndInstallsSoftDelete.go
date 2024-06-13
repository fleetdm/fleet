package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240612150059, Down_20240612150059)
}

func Up_20240612150059(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_script_results ADD COLUMN host_deleted_at TIMESTAMP NULL`)
	if err != nil {
		return fmt.Errorf("failed to add host_deleted_at timestamp to host_script_results: %w", err)
	}
	return nil
}

func Down_20240612150059(tx *sql.Tx) error {
	return nil
}
