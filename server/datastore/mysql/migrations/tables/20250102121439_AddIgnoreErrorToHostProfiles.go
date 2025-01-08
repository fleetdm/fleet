package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250102121439, Down_20250102121439)
}

func Up_20250102121439(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_mdm_apple_profiles
    	ADD COLUMN ignore_error TINYINT(1) NOT NULL DEFAULT 0`)
	if err != nil {
		return fmt.Errorf("failed to add ignore_error to host_mdm_apple_profiles table: %w", err)
	}
	return nil
}

func Down_20250102121439(_ *sql.Tx) error {
	return nil
}
