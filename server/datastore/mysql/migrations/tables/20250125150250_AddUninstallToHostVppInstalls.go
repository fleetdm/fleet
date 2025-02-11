package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250109074626, Down_20250109074626)
}

func Up_20250109074626(tx *sql.Tx) error {

	if !columnExists(tx, "host_vpp_software_installs", "uninstall") {
		if _, err := tx.Exec("ALTER TABLE host_vpp_software_installs ADD COLUMN uninstall TINYINT NOT NULL DEFAULT 0"); err != nil {
			return fmt.Errorf("failed to add removed to host_vpp_software_installs: %w", err)
		}
	}

	return nil
}

func Down_20250109074626(tx *sql.Tx) error {
	return nil
}
