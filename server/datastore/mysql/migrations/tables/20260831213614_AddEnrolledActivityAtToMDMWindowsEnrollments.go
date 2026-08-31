package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831213614, Down_20260831213614)
}

func Up_20260831213614(tx *sql.Tx) error {
	// enrolled_activity_at records when the mdm_enrolled activity was emitted for this enrollment, and doubles as the
	// one-shot claim that keeps it to exactly one activity per enrollment. Azure (automatic) enrollments know neither
	// the host nor its serial at enrollment time, so their activity is emitted later, the first time the enrollment is
	// linked to a host; the claim is what keeps the several code paths that can do that linking from each emitting one.
	//
	// Backfilled to the enrollment's creation time so enrollments that predate this migration are not re-announced.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN enrolled_activity_at DATETIME(6) NULL DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("adding enrolled_activity_at to mdm_windows_enrollments: %w", err)
	}

	if _, err := tx.Exec(`UPDATE mdm_windows_enrollments SET enrolled_activity_at = created_at`); err != nil {
		return fmt.Errorf("backfilling enrolled_activity_at on mdm_windows_enrollments: %w", err)
	}

	return nil
}

func Down_20260831213614(tx *sql.Tx) error {
	return nil
}
