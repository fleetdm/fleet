package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241110152838, Down_20241110152838)
}

func Up_20241110152838(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE software ADD COLUMN team_identifier VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("failed to add team_identifier to software table: %w", err)
	}
	return nil
}

func Down_20241110152838(tx *sql.Tx) error {
	return nil
}
