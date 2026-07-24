package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260724210609, Down_20260724210609)
}

// Up_20260724210609 makes host_managed_local_account_passwords.command_uuid nullable.
// The column records the MDM command that set the password on macOS. Windows accounts are
// created by fleetd on the device and escrowed directly, so their rows have no MDM command
// and store NULL here. The column keeps its index; a nullable column can still be indexed.
func Up_20260724210609(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE host_managed_local_account_passwords " +
			"MODIFY `command_uuid` varchar(127) COLLATE utf8mb4_unicode_ci NULL",
	); err != nil {
		return fmt.Errorf("relaxing host_managed_local_account_passwords.command_uuid to nullable: %w", err)
	}
	return nil
}

func Down_20260724210609(tx *sql.Tx) error {
	return nil
}
