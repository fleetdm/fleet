package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250502154517, Down_20250502154517)
}

func Up_20250502154517(tx *sql.Tx) error {
	if columnExists(tx, "host_mdm_apple_profiles", "variables_updated_at") {
		return nil
	}
	_, err := tx.Exec(`
	ALTER TABLE host_mdm_apple_profiles
	ADD COLUMN variables_updated_at DATETIME(6) NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to add variables_updated_at column to host_mdm_apple_profiles table: %s", err)
	}
	return nil
}

func Down_20250502154517(tx *sql.Tx) error {
	return nil
}
