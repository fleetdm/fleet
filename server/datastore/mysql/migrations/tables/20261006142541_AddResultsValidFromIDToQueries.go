package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261006142541, Down_20261006142541)
}

func Up_20261006142541(tx *sql.Tx) error {
	if columnExists(tx, "queries", "results_valid_from_id") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE queries
		ADD COLUMN results_valid_from_id INT UNSIGNED NOT NULL DEFAULT 0,
		ADD COLUMN results_cleanup_pending TINYINT(1) NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding results_valid_from_id and results_cleanup_pending to queries table: %w", err)
	}
	return nil
}

func Down_20261006142541(tx *sql.Tx) error {
	return nil
}
