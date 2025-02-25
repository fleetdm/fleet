package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20240709183940, Down_20240709183940)
}

// At the time of this migration, all script timeouts are
// hardcoded to 300 seconds. This migration adds a timeout
// column to the host_script_results table to allow for
// custom timeouts.
func Up_20240709183940(tx *sql.Tx) error {
	stmt := `
	ALTER TABLE host_script_results
	ADD COLUMN timeout INT DEFAULT NULL;
	`
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}

	stmt = `
	UPDATE host_script_results
	SET timeout = 300
	WHERE timeout IS NULL;
	`
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}

	return nil
}

func Down_20240709183940(tx *sql.Tx) error {
	return nil
}
