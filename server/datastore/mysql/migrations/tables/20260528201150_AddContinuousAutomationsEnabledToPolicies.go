package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260528201150, Down_20260528201150)
}

func Up_20260528201150(tx *sql.Tx) error {
	if columnExists(tx, "policies", "continuous_automations_enabled") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE policies
		ADD COLUMN continuous_automations_enabled TINYINT(1) NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("add continuous_automations_enabled to policies: %w", err)
	}
	return nil
}

func Down_20260528201150(tx *sql.Tx) error {
	return nil
}
