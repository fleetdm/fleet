package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260522195231, Down_20260522195231)
}

func Up_20260522195231(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE hosts ADD COLUMN orbit_debug_until TIMESTAMP(6) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add orbit_debug_until column to hosts: %w", err)
	}
	return nil
}

func Down_20260522195231(tx *sql.Tx) error {
	return nil
}
