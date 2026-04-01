package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260401220243, Down_20260401220243)
}

func Up_20260401220243(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE nano_commands ADD COLUMN name varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add nano_commands.name column: %w", err)
	}

	return nil
}

func Down_20260401220243(_ *sql.Tx) error {
	return nil
}
