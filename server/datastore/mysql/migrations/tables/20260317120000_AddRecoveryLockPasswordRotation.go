package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260317120000, Down_20260317120000)
}

func Up_20260317120000(tx *sql.Tx) error {
	// Add columns for pending password rotation:
	// - pending_encrypted_password: Holds the new password during rotation until MDM confirms success
	// - pending_error_message: Stores error if rotation fails (separate from main error_message for install/remove)
	if _, err := tx.Exec(`
		ALTER TABLE host_recovery_key_passwords
			ADD COLUMN pending_encrypted_password BLOB DEFAULT NULL,
			ADD COLUMN pending_error_message TEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("adding pending rotation columns to host_recovery_key_passwords: %w", err)
	}
	return nil
}

func Down_20260317120000(tx *sql.Tx) error {
	return nil
}
