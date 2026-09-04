package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260902143818, Down_20260902143818)
}

// Up_20260902143818 adds mdm_windows_enrollments.managed_local_account_rotation_requested, which asks a Windows host to
// re-provision its managed local account so fleetd generates and escrows a new password.
//
// It lives on the enrollment row rather than on host_managed_local_account_passwords so the orbit config check-in, which
// already reads this row once per poll, stays a single indexed lookup. Re-enrolling replaces the row, which clears an
// outstanding request along with the escrowed flag next to it.
func Up_20260902143818(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE mdm_windows_enrollments " +
			"ADD COLUMN `managed_local_account_rotation_requested` tinyint(1) NOT NULL DEFAULT '0'",
	); err != nil {
		return fmt.Errorf("adding mdm_windows_enrollments.managed_local_account_rotation_requested: %w", err)
	}
	return nil
}

func Down_20260902143818(tx *sql.Tx) error {
	return nil
}
