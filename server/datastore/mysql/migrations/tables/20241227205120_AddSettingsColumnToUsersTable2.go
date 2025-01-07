package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241227205120, Down_20241227205120)
}

func Up_20241227205120(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE users ADD COLUMN settings json NOT NULL DEFAULT ('{}')`)
	if err != nil {
		return fmt.Errorf("failed to add settings to users: %w", err)
	}
	return nil
}

func Down_20241227205120(tx *sql.Tx) error {
	return nil
}
