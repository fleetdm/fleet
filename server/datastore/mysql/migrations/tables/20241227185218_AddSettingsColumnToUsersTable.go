package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241227185218, Down_20241227185218)
}

func Up_20241227185218(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE users ADD COLUMN settings json`)
	if err != nil {
		return fmt.Errorf("failed to add settings to users: %w", err)
	}
	return nil
}

func Down_20241227185218(tx *sql.Tx) error {
	return nil
}
