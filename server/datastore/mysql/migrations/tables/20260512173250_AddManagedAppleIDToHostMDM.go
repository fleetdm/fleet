package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260512173250, Down_20260512173250)
}

func Up_20260512173250(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_mdm
		ADD COLUMN managed_apple_id VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER fleet_enroll_ref
	`); err != nil {
		return fmt.Errorf("adding managed_apple_id to host_mdm: %w", err)
	}
	return nil
}

func Down_20260512173250(tx *sql.Tx) error {
	return nil
}
