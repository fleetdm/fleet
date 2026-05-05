package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260326131501, Down_20260326131501)
}

func Up_20260326131501(tx *sql.Tx) error {
	// Add auto_rotate_at column to track when a viewed password should be automatically rotated.
	// When a password is viewed via the API, auto_rotate_at is set to 1 hour in the future.
	// The cron job rotates passwords where auto_rotate_at <= NOW().
	if _, err := tx.Exec(`
		ALTER TABLE host_recovery_key_passwords
			ADD COLUMN auto_rotate_at TIMESTAMP(6) NULL DEFAULT NULL,
			ADD INDEX idx_auto_rotate_at (auto_rotate_at)
	`); err != nil {
		return fmt.Errorf("adding auto_rotate_at column to host_recovery_key_passwords: %w", err)
	}
	return nil
}

func Down_20260326131501(tx *sql.Tx) error {
	return nil
}
