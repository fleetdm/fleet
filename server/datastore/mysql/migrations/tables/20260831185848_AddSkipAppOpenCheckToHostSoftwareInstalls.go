package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831185848, Down_20260831185848)
}

func Up_20260831185848(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "skip_app_open_check") {
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN skip_app_open_check TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add skip_app_open_check to host_software_installs: %w", err)
		}
	}

	return nil
}

func Down_20260831185848(tx *sql.Tx) error {
	return nil
}
