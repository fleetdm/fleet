package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240617120944, Down_20240617120944)
}

func Up_20240617120944(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD COLUMN timezone TEXT NULL`); err != nil {
		return fmt.Errorf("failed to add `timezone` column to `calendar_events` table: %w", err)
	}
	return nil
}

func Down_20240617120944(tx *sql.Tx) error {
	return nil
}
