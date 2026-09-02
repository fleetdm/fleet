package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260902140128, Down_20260902140128)
}

func Up_20260902140128(tx *sql.Tx) error {
	// enrolled_activity_at records when the mdm_enrolled activity was emitted for this enrollment
	//
	// Every enrollment that predates this column was already announced by the old code, so all of them must start out
	// non-NULL. The column is added WITH a default so pre-existing rows read as backfilled without rewriting them.
	// The value matches common_mysql.DefaultNonZeroTime, the same marker the service writes when it resolves an
	// enrollment without recording anything.
	if _, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN enrolled_activity_at DATETIME(6) NULL DEFAULT '2000-01-01 00:00:00.000000'
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

func Down_20260902140128(tx *sql.Tx) error {
	return nil
}
