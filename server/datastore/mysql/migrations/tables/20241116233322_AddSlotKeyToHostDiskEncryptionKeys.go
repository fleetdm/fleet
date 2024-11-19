package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241116233322, Down_20241116233322)
}

func Up_20241116233322(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_disk_encryption_keys ADD COLUMN base64_encrypted_slot_key VARCHAR(255) NOT NULL DEFAULT '' AFTER base64_encrypted`)
	if err != nil {
		return fmt.Errorf("failed to add base64_encrypted_slot_key to host_disk_encryption_keys: %w", err)
	}

	return nil
}

func Down_20241116233322(tx *sql.Tx) error {
	return nil
}
