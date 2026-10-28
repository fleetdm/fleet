package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261027212007, Down_20261027212007)
}

// Up_20261027212007 clears the escrowed flag on Windows enrollments whose host has no live managed local account
// password under its current UUID. An error-only row has no password, so it does not count. Clearing the flag makes
// fleetd re-create the account on its next config fetch, which resets the password and escrows it under the current
// UUID. Enrollments of deleted hosts are included so a host that comes back with its enrollment recovers too.
func Up_20261027212007(tx *sql.Tx) error {
	// updated_at is kept because the stale enrollment cleanup keys off it.
	if _, err := tx.Exec(`
		UPDATE mdm_windows_enrollments e
		SET e.managed_local_account_escrowed = 0, e.updated_at = e.updated_at
		WHERE e.managed_local_account_escrowed = 1
			AND NOT EXISTS (
				SELECT 1 FROM host_managed_local_account_passwords p
				WHERE p.host_uuid = e.host_uuid AND p.deleted = 0 AND LENGTH(p.encrypted_password) > 0
			)`); err != nil {
		return fmt.Errorf("clearing stranded windows managed local account escrowed flags: %w", err)
	}
	return nil
}

func Down_20261027212007(tx *sql.Tx) error {
	return nil
}
