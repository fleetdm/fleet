package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260909183250, Down_20260909183250)
}

func Up_20260909183250(tx *sql.Tx) error {
	// NULL means the host has not reported yet, which is the state every existing row starts in and the state a host
	// running an agent without the reporting query stays in. Will be set by the next detailed query.
	_, err := tx.Exec(`ALTER TABLE host_disks ADD COLUMN bitlocker_boot_protector_set TINYINT(1) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding bitlocker_boot_protector_set to host_disks: %w", err)
	}
	return nil
}

func Down_20260909183250(tx *sql.Tx) error {
	return nil
}
