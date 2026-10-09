package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261007183944, Down_20261007183944)
}

func Up_20261007183944(tx *sql.Tx) error {
	if columnExists(tx, "mdm_windows_enrollments", "fleetd_present_at") {
		return nil
	}
	// Existing rows start NULL, so their next management session re-checks fleetd presence once and records it.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN fleetd_present_at DATETIME(6) NULL DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("adding fleetd_present_at to mdm_windows_enrollments: %w", err)
	}
	return nil
}

func Down_20261007183944(tx *sql.Tx) error {
	return nil
}
