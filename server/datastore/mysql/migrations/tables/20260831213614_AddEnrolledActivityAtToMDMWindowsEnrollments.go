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
	// Every enrollment that predates this column was already announced by the old code, so all of them must start out
	// non-NULL or the next management session would re-announce every Windows host in the fleet. The column is added
	// WITH a default so pre-existing rows read as backfilled without rewriting them: mdm_windows_enrollments holds a
	// row per Windows MDM enrollment and scales with host count, and a single unbounded UPDATE over it can stall an
	// upgrade. The default is then set to NULL so new enrollments are inserted unclaimed. It must be SET DEFAULT NULL
	// rather than DROP DEFAULT: dropping it leaves the column with no default at all, and strict mode then rejects
	// every insert that does not name it, which the enrollment insert does not.
	// The default value itself is never read: only whether enrolled_activity_at is NULL decides whether the
	// mdm_enrolled activity still has to be recorded.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN enrolled_activity_at DATETIME(6) NULL DEFAULT '1970-01-01 00:00:00.000000'
	`); err != nil {
		return fmt.Errorf("adding enrolled_activity_at to mdm_windows_enrollments: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ALTER COLUMN enrolled_activity_at SET DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("resetting default for enrolled_activity_at on mdm_windows_enrollments: %w", err)
	}

	return nil
}

func Down_20260831213614(tx *sql.Tx) error {
	return nil
}
