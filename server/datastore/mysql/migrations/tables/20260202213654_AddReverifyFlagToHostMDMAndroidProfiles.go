package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260202213654, Down_20260202213654)
}

func Up_20260202213654(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_mdm_android_profiles ADD COLUMN reverify tinyint(1) NOT NULL DEFAULT '0'`)
	if err != nil {
		return fmt.Errorf("failed to add reverify to host_mdm_android_profiles: %w", err)
	}
	return nil
}

func Down_20260202213654(tx *sql.Tx) error {
	return nil
}
