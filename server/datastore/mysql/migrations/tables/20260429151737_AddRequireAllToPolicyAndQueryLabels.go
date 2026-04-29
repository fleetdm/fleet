package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260429151737, Down_20260429151737)
}

func Up_20260429151737(tx *sql.Tx) error {
	if !columnExists(tx, "policy_labels", "require_all") {
		if _, err := tx.Exec(`
			ALTER TABLE policy_labels
			ADD COLUMN require_all TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("failed to add require_all to policy_labels: %w", err)
		}
	}
	if !columnExists(tx, "query_labels", "require_all") {
		if _, err := tx.Exec(`
			ALTER TABLE query_labels
			ADD COLUMN require_all TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("failed to add require_all to query_labels: %w", err)
		}
	}
	return nil
}

func Down_20260429151737(tx *sql.Tx) error {
	return nil
}
