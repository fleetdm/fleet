package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260818162738, Down_20260818162738)
}

func Up_20260818162738(tx *sql.Tx) error {
	if !columnExists(tx, "policies", "notify_before_patching") {
		if _, err := tx.Exec(`
			ALTER TABLE policies
			ADD COLUMN notify_before_patching TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add notify_before_patching to policies: %w", err)
		}
	}

	return nil
}

func Down_20260818162738(tx *sql.Tx) error {
	return nil
}
