package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240815000001, Down_20240815000001)
}

func Up_20240815000001(tx *sql.Tx) error {
	if _, err := tx.Exec("ALTER TABLE vpp_apps_teams ADD COLUMN self_service bool NOT NULL DEFAULT false"); err != nil {
		return fmt.Errorf("Failed to add self_service to vpp_apps_teams: %w", err)
	}
	return nil
}

func Down_20240815000001(tx *sql.Tx) error {
	return nil
}
