package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260922124402, Down_20260922124402)
}

func Up_20260922124402(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE policies ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("adding hidden to policies table: %w", err)
	}
	return nil
}

func Down_20260922124402(tx *sql.Tx) error {
	return nil
}
