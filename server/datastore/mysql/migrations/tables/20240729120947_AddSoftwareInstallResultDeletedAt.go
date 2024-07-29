package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240729120947, Down_20240729120947)
}

func Up_20240729120947(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE host_software_installs ADD COLUMN host_deleted_at timestamp NULL DEFAULT NULL")
	if err != nil {
		return fmt.Errorf("failed to create host_deleted_at column on host_software_installs table: %w", err)
	}
	return nil
}

func Down_20240729120947(tx *sql.Tx) error {
	return nil
}
