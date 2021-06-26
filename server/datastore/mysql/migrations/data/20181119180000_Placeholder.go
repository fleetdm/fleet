package data

import (
	"database/sql"
)

// This migration exists as a "placeholder" to prevent the unexpected printing
// of a migration inconsistency warning on Fleet startup. The migration is
// intentionally empty. See https://github.com/fleetdm/fleet/v4/issues/48 for more
// details.

func init() {
	MigrationClient.AddMigration(Up_20181119180000, Down_20181119180000)
}

func Up_20181119180000(tx *sql.Tx) error {
	// Intentional noop
	return nil
}

func Down_20181119180000(tx *sql.Tx) error {
	// Intentional noop
	return nil
}
