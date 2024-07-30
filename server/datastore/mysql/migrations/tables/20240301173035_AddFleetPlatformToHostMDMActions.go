package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240301173035, Down_20240301173035)
}

func Up_20240301173035(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE host_mdm_actions
	ADD COLUMN fleet_platform VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_mdm_actions table: %w", err)
	}
	return nil
}

func Down_20240301173035(tx *sql.Tx) error {
	return nil
}
