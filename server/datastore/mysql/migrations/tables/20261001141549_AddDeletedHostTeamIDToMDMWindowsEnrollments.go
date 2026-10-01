package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261001141549, Down_20261001141549)
}

// Up_20261001141549 records the fleet of a Windows MDM host when it is deleted, so the one-time enroll secret Fleet pushes to its
// enrollment brings it back to that fleet.
func Up_20261001141549(tx *sql.Tx) error {
	if !columnExists(tx, "mdm_windows_enrollments", "deleted_host_team_id") {
		if _, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments ADD COLUMN deleted_host_team_id INT UNSIGNED NULL`); err != nil {
			return fmt.Errorf("adding deleted_host_team_id to mdm_windows_enrollments: %w", err)
		}
	}
	return nil
}

func Down_20261001141549(tx *sql.Tx) error {
	return nil
}
