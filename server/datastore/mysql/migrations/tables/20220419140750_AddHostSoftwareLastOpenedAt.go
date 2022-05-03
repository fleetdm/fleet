package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20220419140750, Down_20220419140750)
}

func Up_20220419140750(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `host_software` ADD COLUMN `last_opened_at` timestamp NULL",
	)
	if err != nil {
		return fmt.Errorf("add last_opened_at column: %w", err)
	}

	return nil
}

func Down_20220419140750(tx *sql.Tx) error {
	return nil
}
