package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250219090511, Down_20250219090511)
}

func Up_20250219090511(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE activities ADD COLUMN fleet_initiated tinyint(1) NOT NULL DEFAULT '0'`)
	if err != nil {
		return fmt.Errorf("failed to add fleet_initiated to activities: %w", err)
	}
	return nil
}

func Down_20250219090511(tx *sql.Tx) error {
	return nil
}
