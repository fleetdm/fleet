package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260909231555, Down_20260909231555)
}

// escrow_sent_at marks when a LUKS escrow request was handed to fleetd; cleared when fleetd reports a result.
func Up_20260909231555(tx *sql.Tx) error {
	if columnExists(tx, "host_disk_encryption_keys", "escrow_sent_at") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE host_disk_encryption_keys
		ADD COLUMN escrow_sent_at TIMESTAMP(6) NULL DEFAULT NULL
	`); err != nil {
		return fmt.Errorf("adding escrow_sent_at to host_disk_encryption_keys: %w", err)
	}
	return nil
}

func Down_20260909231555(tx *sql.Tx) error {
	return nil
}
