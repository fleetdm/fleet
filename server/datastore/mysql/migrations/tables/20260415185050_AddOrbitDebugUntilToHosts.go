package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260415185050, Down_20260415185050)
}

func Up_20260415185050(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE hosts ADD COLUMN orbit_debug_until TIMESTAMP NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add orbit_debug_until column to hosts: %w", err)
	}
	return nil
}

func Down_20260415185050(tx *sql.Tx) error {
	return nil
}
