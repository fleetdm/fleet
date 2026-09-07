package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260907212114, Down_20260907212114)
}

// Up_20260907212114 adds mdm_windows_enrollments.managed_local_account_rotation_requested. It lives on the enrollment
// row so the per-poll orbit config read stays a single lookup, and re-enrollment clears a stale request for free.
func Up_20260907212114(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE mdm_windows_enrollments " +
			"ADD COLUMN `managed_local_account_rotation_requested` tinyint(1) NOT NULL DEFAULT '0'",
	); err != nil {
		return fmt.Errorf("adding mdm_windows_enrollments.managed_local_account_rotation_requested: %w", err)
	}
	return nil
}

func Down_20260907212114(tx *sql.Tx) error {
	return nil
}
