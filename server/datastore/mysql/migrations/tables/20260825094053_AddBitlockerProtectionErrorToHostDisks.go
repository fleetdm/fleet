package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260825094053, Down_20260825094053)
}

func Up_20260825094053(tx *sql.Tx) error {
	// Holds the reason the agent could not restore BitLocker protection, so the host's disk encryption detail can state
	// what actually happened instead of inferring it from protection status. Kept out of host_disk_encryption_keys on
	// purpose: writes there also own base64_encrypted, and reporting an error through that path would blank the
	// escrowed recovery key on exactly the hosts that still need it.
	if _, err := tx.Exec(`ALTER TABLE host_disks ADD COLUMN bitlocker_protection_error VARCHAR(255) NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("adding bitlocker_protection_error to host_disks: %w", err)
	}
	return nil
}

func Down_20260825094053(tx *sql.Tx) error {
	return nil
}
