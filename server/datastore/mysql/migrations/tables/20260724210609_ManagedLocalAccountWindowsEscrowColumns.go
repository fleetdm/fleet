package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260724210609, Down_20260724210609)
}

// Up_20260724210609 adapts host_managed_local_account_passwords to accounts created by fleetd on
// the device (Windows) rather than by an MDM command:
//
//   - command_uuid becomes nullable. It records the MDM command that set the password on macOS;
//     Windows accounts have no such command and store NULL. The column keeps its index; a nullable
//     column can still be indexed.
//   - encrypted_password becomes nullable, so a row that only records a failed creation attempt can
//     say it holds no password instead of holding an empty blob that is not valid ciphertext. The
//     table's existing "encrypted_password IS NOT NULL" guards were written for this and are
//     tautologies until the column is nullable.
//   - client_error is added to record the device-reported reason the account could not be created,
//     mirroring host_disk_encryption_keys.client_error.
func Up_20260724210609(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE host_managed_local_account_passwords " +
			"MODIFY `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NULL, " +
			"MODIFY `encrypted_password` blob NULL, " +
			"ADD COLUMN `client_error` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''",
	); err != nil {
		return fmt.Errorf("adapting host_managed_local_account_passwords for device-created accounts: %w", err)
	}
	return nil
}

func Down_20260724210609(tx *sql.Tx) error {
	return nil
}
