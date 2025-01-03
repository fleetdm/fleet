package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250103112935, Down_20250103112935)
}

func Up_20250103112935(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE users ADD COLUMN settings json NOT NULL DEFAULT (JSON_OBJECT())`)
	if err != nil {
		return fmt.Errorf("failed to add settings to users: %w", err)
	}
	return nil
}

func Down_20250103112935(tx *sql.Tx) error {
	return nil
}
