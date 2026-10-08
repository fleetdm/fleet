package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261006131520, Down_20261006131520)
}

// Up_20261006131520 records the Entra device ID signed into an Entra enrollment's access token, so a later enrollment presenting
// the same hardware ID can be checked against the device that holds it.
func Up_20261006131520(tx *sql.Tx) error {
	if !columnExists(tx, "mdm_windows_enrollments", "entra_device_id") {
		if _, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments ADD COLUMN entra_device_id BINARY(16) NULL`); err != nil {
			return fmt.Errorf("adding entra_device_id to mdm_windows_enrollments: %w", err)
		}
	}
	return nil
}

func Down_20261006131520(tx *sql.Tx) error {
	return nil
}
