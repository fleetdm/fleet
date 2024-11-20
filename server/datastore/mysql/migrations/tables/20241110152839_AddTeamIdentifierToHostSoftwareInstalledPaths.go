package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241110152839, Down_20241110152839)
}

func Up_20241110152839(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_software_installed_paths ADD COLUMN team_identifier VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("failed to add team_identifier to host_software_installed_paths table: %w", err)
	}
	return nil
}

func Down_20241110152839(tx *sql.Tx) error {
	return nil
}
