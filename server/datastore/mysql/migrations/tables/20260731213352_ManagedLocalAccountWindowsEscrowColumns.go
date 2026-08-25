package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260731213352, Down_20260731213352)
}

// Up_20260731213352 adapts host_managed_local_account_passwords to accounts created by fleetd on
// the device (Windows):
//   - command_uuid becomes nullable. It records the MDM command that set the password on macOS;
//   - encrypted_password becomes nullable, so a row that only records a failed creation attempt can
//     say it holds no password.
//   - client_error is added to record the device-reported reason the account could not be created,
//     mirroring host_disk_encryption_keys.client_error.
//
// It also adds mdm_windows_enrollments.managed_local_account_escrowed, which records that the device
// has escrowed a password for its current enrollment.
func Up_20260731213352(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE host_managed_local_account_passwords " +
			"MODIFY `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NULL, " +
			"MODIFY `encrypted_password` blob NULL, " +
			"ADD COLUMN `client_error` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''",
	); err != nil {
		return fmt.Errorf("adapting host_managed_local_account_passwords for device-created accounts: %w", err)
	}

	if _, err := tx.Exec(
		"ALTER TABLE mdm_windows_enrollments " +
			"ADD COLUMN `managed_local_account_escrowed` tinyint(1) NOT NULL DEFAULT '0'",
	); err != nil {
		return fmt.Errorf("adding mdm_windows_enrollments.managed_local_account_escrowed: %w", err)
	}
	return nil
}

func Down_20260731213352(tx *sql.Tx) error {
	return nil
}
