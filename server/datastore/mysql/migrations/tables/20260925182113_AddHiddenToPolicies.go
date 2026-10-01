package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260925182113, Down_20260925182113)
}

func Up_20260925182113(tx *sql.Tx) error {
	if columnExists(tx, "policies", "hidden") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE policies
		ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding hidden to policies table: %w", err)
	}
	return nil
}

func Down_20260925182113(tx *sql.Tx) error {
	return nil
}
