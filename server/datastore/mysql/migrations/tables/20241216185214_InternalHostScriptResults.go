package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241216185214, Down_20241216185214)
}

func Up_20241216185214(tx *sql.Tx) error {
	// alter the host_script_results table to add an is_internal flag (for scripts that still execute
	// when scripts are disabled globally)
	_, err := tx.Exec(`ALTER TABLE host_script_results ADD COLUMN is_internal BOOLEAN DEFAULT FALSE`)
	if err != nil {
		return fmt.Errorf("add is_internal to host_script_results: %w", err)
	}

	return nil
}

func Down_20241216185214(tx *sql.Tx) error {
	return nil
}
