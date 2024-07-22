package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240626195531, Down_20240626195531)
}

func Up_20240626195531(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD COLUMN timezone VARCHAR(64) COLLATE utf8mb4_unicode_ci NULL`); err != nil {
		return fmt.Errorf("failed to add `timezone` column to `calendar_events` table: %w", err)
	}
	return nil
}

func Down_20240626195531(tx *sql.Tx) error {
	return nil
}
