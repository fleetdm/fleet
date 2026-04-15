package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260415200000, Down_20260415200000)
}

func Up_20260415200000(tx *sql.Tx) error {
	// Change the Android builtin label from manual membership (with no query)
	// to dynamic membership with a platform-based query, matching the pattern
	// used by the Chrome builtin label. This ensures Android hosts enrolled
	// via the osquery protocol are automatically added to the label.
	_, err := tx.Exec(`
		UPDATE labels
		SET query = 'select 1 from os_version where platform = ''android'';',
		    label_membership_type = 0
		WHERE name = 'Android' AND label_type = 1
	`)
	return err
}

func Down_20260415200000(tx *sql.Tx) error {
	return nil
}
