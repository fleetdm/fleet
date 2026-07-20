package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260720174112, Down_20260720174112)
}

func Up_20260720174112(tx *sql.Tx) error {
	if !columnExists(tx, "policies", "patch_when_closed") {
		if _, err := tx.Exec(`
			ALTER TABLE policies
			ADD COLUMN patch_when_closed TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add patch_when_closed to policies: %w", err)
		}
	}

	if !columnExists(tx, "software_installers", "app_open_query") {
		if _, err := tx.Exec(`
			ALTER TABLE software_installers
			ADD COLUMN app_open_query TEXT COLLATE utf8mb4_unicode_ci NOT NULL
		`); err != nil {
			return fmt.Errorf("add app_open_query to software_installers: %w", err)
		}
	}

	return nil
}

func Down_20260720174112(tx *sql.Tx) error {
	return nil
}
