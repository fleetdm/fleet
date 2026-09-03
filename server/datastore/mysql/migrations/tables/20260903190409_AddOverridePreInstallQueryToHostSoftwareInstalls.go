package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260903190409, Down_20260903190409)
}

func Up_20260903190409(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "override_pre_install_query") {
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN override_pre_install_query TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add override_pre_install_query to host_software_installs: %w", err)
		}
	}

	return nil
}

func Down_20260903190409(tx *sql.Tx) error {
	return nil
}
