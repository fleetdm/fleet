package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260910194211, Down_20260910194211)
}

func Up_20260910194211(tx *sql.Tx) error {
	if columnExists(tx, "host_disks", "bitlocker_boot_protector_set") {
		return nil
	}
	// NULL means the host has not reported yet, which is the state every existing row starts in and the state a host
	// running an agent without the reporting query stays in. Will be set by the next detailed query.
	if _, err := tx.Exec(`
		ALTER TABLE host_disks
		ADD COLUMN bitlocker_boot_protector_set TINYINT(1) NULL DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("adding bitlocker_boot_protector_set to host_disks: %w", err)
	}
	return nil
}

func Down_20260910194211(tx *sql.Tx) error {
	return nil
}
