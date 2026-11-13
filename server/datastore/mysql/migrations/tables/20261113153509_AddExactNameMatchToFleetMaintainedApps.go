package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261113153509, Down_20261113153509)
}

// Up_20261113153509 adds exact_name_match, which the apps list sync fills in. Until it
// does, every app keeps the prefix name matching it had before.
func Up_20261113153509(tx *sql.Tx) error {
	if !columnExists(tx, "fleet_maintained_apps", "exact_name_match") {
		if _, err := tx.Exec(`ALTER TABLE fleet_maintained_apps
			ADD COLUMN exact_name_match tinyint(1) NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add exact_name_match column to fleet_maintained_apps: %w", err)
		}
	}
	return nil
}

func Down_20261113153509(tx *sql.Tx) error {
	return nil
}
