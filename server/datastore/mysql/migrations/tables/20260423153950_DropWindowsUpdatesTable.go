package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260423153950, Down_20260423153950)
}

func Up_20260423153950(tx *sql.Tx) error {
	if _, err := tx.Exec(`DROP TABLE IF EXISTS windows_updates`); err != nil {
		return fmt.Errorf("drop windows_updates table: %w", err)
	}
	return nil
}

func Down_20260423153950(tx *sql.Tx) error {
	return nil
}
