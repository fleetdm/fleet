package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration("Add the throttled retry time to host DEP assignments", Up_20261124032357, Down_20261124032357)
}

// Up_20261124032357 adds when a throttled profile assignment may be retried, as asked by Apple. Rows
// throttled before this, or without Apple's retry_after_seconds, stay NULL and keep the fixed cooldown.
func Up_20261124032357(tx *sql.Tx) error {
	if columnExists(tx, "host_dep_assignments", "throttled_retry_after") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE host_dep_assignments
		ADD COLUMN throttled_retry_after TIMESTAMP NULL DEFAULT NULL AFTER response_updated_at`); err != nil {
		return fmt.Errorf("adding throttled_retry_after to host_dep_assignments: %w", err)
	}
	return nil
}

func Down_20261124032357(tx *sql.Tx) error {
	return nil
}
