package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250428145750, Down_20250428145750)
}

func Up_20250428145750(tx *sql.Tx) error {
	if columnExists(tx, "host_mdm_managed_certificates", "install_requested_at") {
		return nil
	}
	_, err := tx.Exec(`
	ALTER TABLE host_mdm_managed_certificates
	ADD COLUMN install_requested_at timestamp(6) NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to add install_requested_at column to host_mdm_managed_certificates table: %s", err)
	}
	return nil
}

func Down_20250428145750(tx *sql.Tx) error {
	return nil
}
