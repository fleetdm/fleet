package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250729161235, Down_20250729161235)
}

func Up_20250729161235(tx *sql.Tx) error {
	if _, err := tx.Exec(`
			ALTER TABLE host_disks
			ADD COLUMN tpm_pin_set bool DEFAULT false
		`); err != nil {
		return fmt.Errorf("failed to add 'tpm_pin_set' column to 'host_disks': %w", err)
	}
	return nil
}

func Down_20250729161235(tx *sql.Tx) error {
	return nil
}
