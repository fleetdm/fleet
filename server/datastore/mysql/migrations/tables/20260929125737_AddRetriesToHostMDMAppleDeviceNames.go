package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260929125737, Down_20260929125737)
}

func Up_20260929125737(tx *sql.Tx) error {
	if columnExists(tx, "host_mdm_apple_device_names", "retries") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE host_mdm_apple_device_names
		ADD COLUMN retries TINYINT UNSIGNED NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding retries to host_mdm_apple_device_names table: %w", err)
	}
	return nil
}

func Down_20260929125737(tx *sql.Tx) error {
	return nil
}
