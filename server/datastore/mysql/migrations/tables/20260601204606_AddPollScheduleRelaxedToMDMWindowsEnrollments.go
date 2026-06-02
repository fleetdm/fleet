package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260601204606, Down_20260601204606)
}

func Up_20260601204606(tx *sql.Tx) error {
	// poll_schedule_relaxed tracks whether the enrollment's DMClient poll schedule has been relaxed because the host's fleetd can be woken on
	// demand (Windows MDM sync). The management session reconciles the schedule against this so it does not re-send the poll Replace on every
	// session.
	if _, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments
		ADD COLUMN poll_schedule_relaxed TINYINT(1) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add poll_schedule_relaxed to mdm_windows_enrollments: %w", err)
	}
	return nil
}

func Down_20260601204606(tx *sql.Tx) error {
	return nil
}
