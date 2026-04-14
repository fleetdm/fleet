package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260414172920, Down_20260414172920)
}

func Up_20260414172920(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE mdm_config_assets ADD COLUMN renew_at TIMESTAMP NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding renew_at to mdm_config_assets: %w", err)
	}
	return nil
}

func Down_20260414172920(tx *sql.Tx) error {
	return nil
}
