package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260826205421, Down_20260826205421)
}

func Up_20260826205421(tx *sql.Tx) error {
	// Holds the reason the agent could not restore BitLocker protection, so the host's disk encryption detail can state
	// what actually happened.
	if _, err := tx.Exec(`ALTER TABLE host_disks ADD COLUMN bitlocker_protection_error VARCHAR(255) NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("adding bitlocker_protection_error to host_disks: %w", err)
	}
	return nil
}

func Down_20260826205421(tx *sql.Tx) error {
	return nil
}
