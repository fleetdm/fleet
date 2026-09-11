package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260901192408, Down_20260901192408)
}

func Up_20260901192408(tx *sql.Tx) error {
	// bitlocker_protection_error holds the reason the agent could not restore BitLocker protection, and
	// bitlocker_protection_outcome separates a repair it deliberately deferred, which resolves itself, from one that
	// failed. Together they let the host's disk encryption detail state what happened and name what has to happen next.
	if _, err := tx.Exec(`ALTER TABLE host_disks
		ADD COLUMN bitlocker_protection_error VARCHAR(255)
			CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
		ADD COLUMN bitlocker_protection_outcome ENUM('deferred','failed')
			CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("adding bitlocker protection columns to host_disks: %w", err)
	}
	return nil
}

func Down_20260901192408(tx *sql.Tx) error {
	return nil
}
