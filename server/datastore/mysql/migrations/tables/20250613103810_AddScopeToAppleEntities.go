package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250613103810, Down_20250613103810)
}

func Up_20250613103810(tx *sql.Tx) error {
	if !columnExists(tx, "mdm_apple_configuration_profiles", "scope") {
		_, err := tx.Exec(`
		ALTER TABLE mdm_apple_configuration_profiles
		ADD COLUMN scope ENUM('System', 'User') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'System'
	`)
		if err != nil {
			return fmt.Errorf("failed to add scope to mdm_apple_configuration_profiles table: %w", err)
		}
	}
	if !columnExists(tx, "mdm_apple_declarations", "scope") {
		_, err := tx.Exec(`
		ALTER TABLE mdm_apple_declarations
		ADD COLUMN scope ENUM('System', 'User') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'System'
	`)
		if err != nil {
			return fmt.Errorf("failed to add scope to mdm_apple_declarations table: %w", err)
		}
	}

	if !columnExists(tx, "host_mdm_apple_profiles", "scope") {
		_, err := tx.Exec(`
		ALTER TABLE host_mdm_apple_profiles
		ADD COLUMN scope ENUM('System', 'User') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'System'
	`)
		if err != nil {
			return fmt.Errorf("failed to add scope to host_mdm_apple_profiles table: %w", err)
		}
	}
	if !columnExists(tx, "host_mdm_apple_declarations", "scope") {
		_, err := tx.Exec(`
		ALTER TABLE host_mdm_apple_declarations
		ADD COLUMN scope ENUM('System', 'User') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'System'
	`)
		if err != nil {
			return fmt.Errorf("failed to add scope to host_mdm_apple_declarations table: %w", err)
		}
	}

	return nil
}

func Down_20250613103810(tx *sql.Tx) error {
	return nil
}
