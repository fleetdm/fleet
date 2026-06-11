package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260409183610, Down_20260409183610)
}

func Up_20260409183610(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_disks ADD COLUMN bitlocker_protection_status TINYINT(1) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding bitlocker_protection_status to host_disks: %w", err)
	}
	return nil
}

func Down_20260409183610(tx *sql.Tx) error {
	return nil
}
