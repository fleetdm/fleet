package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261127104416, Down_20261127104416)
}

// Up_20261127104416 lets a script result be hidden from the host's current script status while staying
// readable from its activity, like host_software_installs.removed.
func Up_20261127104416(tx *sql.Tx) error {
	if columnExists(tx, "host_script_results", "removed") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE host_script_results
		ADD COLUMN removed TINYINT(1) NOT NULL DEFAULT '0' AFTER canceled`); err != nil {
		return fmt.Errorf("adding removed to host_script_results: %w", err)
	}
	return nil
}

func Down_20261127104416(tx *sql.Tx) error {
	return nil
}
