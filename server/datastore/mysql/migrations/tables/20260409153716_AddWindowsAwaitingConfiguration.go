package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260409153716, Down_20260409153716)
}

func Up_20260409153716(tx *sql.Tx) error {
	if columnExists(tx, "mdm_windows_enrollments", "awaiting_configuration") {
		return nil
	}
	_, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN awaiting_configuration TINYINT(1) NOT NULL DEFAULT 0,
		ADD COLUMN awaiting_configuration_at DATETIME(6) DEFAULT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to add awaiting_configuration columns to mdm_windows_enrollments: %w", err)
	}
	return nil
}

func Down_20260409153716(tx *sql.Tx) error {
	return nil
}
