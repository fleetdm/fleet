package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240821160025, Down_20240821160025)
}

func Up_20240821160025(tx *sql.Tx) error {
	if columnExists(tx, "host_software_installs", "removed") {
		return nil
	}

	if _, err := tx.Exec("ALTER TABLE host_software_installs ADD COLUMN removed TINYINT NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to add removed to host_software_installs: %w", err)
	}

	return nil
}

func Down_20240821160025(_ *sql.Tx) error {
	return nil
}
