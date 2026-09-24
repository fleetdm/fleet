package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260810192005, Down_20260810192005)
}

func Up_20260810192005(tx *sql.Tx) error {
	if _, err := tx.Exec(`UPDATE software_installers SET extension = LOWER(extension)`); err != nil {
		return fmt.Errorf("lowercasing software installer extensions: %w", err)
	}

	return nil
}

func Down_20260810192005(tx *sql.Tx) error {
	return nil
}
