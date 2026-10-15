package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261015175850, Down_20261015175850)
}

func Up_20261015175850(tx *sql.Tx) error {
	// The Autopilot sync matches registry devices to enrolled hosts through the IDs on their Windows MDM enrollment, and orbit
	// enrollment looks up the unlinked enrollment carrying the Autopilot device ID it reports.
	for _, index := range []struct{ name, column string }{
		{"idx_mdm_windows_enrollments_ztd_registration_id", "ztd_registration_id"},
		{"idx_mdm_windows_enrollments_entra_device_id", "entra_device_id"},
	} {
		if indexExistsTx(tx, "mdm_windows_enrollments", index.name) {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`ALTER TABLE mdm_windows_enrollments ADD KEY %s (%s)`, index.name, index.column)); err != nil {
			return fmt.Errorf("add %s: %w", index.name, err)
		}
	}
	return nil
}

func Down_20261015175850(tx *sql.Tx) error {
	return nil
}
