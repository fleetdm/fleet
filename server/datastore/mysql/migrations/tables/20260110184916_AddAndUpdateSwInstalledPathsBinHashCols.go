package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260110184916, Down_20260110184916)
}

func Up_20260110184916(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE host_software_installed_paths
	ADD COLUMN binary_sha256 CHAR(64) COLLATE utf8mb4_unicode_ci NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to add column 'binary_sha256' to 'host_software_installed_paths' table: %w", err)
	}

	_, err = tx.Exec(`
	ALTER TABLE host_software_installed_paths
	CHANGE executable_sha256 cdhash_sha256 CHAR(64) COLLATE utf8mb4_unicode_ci NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to update name of 'host_software_installed_paths' column 'binary_sha256' to `cdhash_sha256`: %w", err)
	}

	return nil
}

func Down_20260110184916(tx *sql.Tx) error {
	return nil
}
