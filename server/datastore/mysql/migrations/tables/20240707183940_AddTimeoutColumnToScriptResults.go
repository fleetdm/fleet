package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20240707183940, Down_20240707183940)
}

func Up_20240707183940(tx *sql.Tx) error {
	// At the time of this migration, all script timeouts are
	// hardcoded to 300 seconds. This migration adds a timeout
	// column to the host_script_results table to allow for
	// custom timeouts.
	stmt := `
	ALTER TABLE host_script_results
	ADD COLUMN timeout INT NOT NULL DEFAULT 300
	`
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}

	stmt = `
	ALTER TABLE host_script_results
	ALTER COLUMN timeout DROP DEFAULT
	`
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}

	return nil
}

func Down_20240707183940(tx *sql.Tx) error {
	return nil
}
