package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250421085116, Down_20250421085116)
}

func Up_20250421085116(tx *sql.Tx) error {
	if columnExists(tx, "host_mdm_managed_certificates", "serial") {
		return nil
	}
	_, err := tx.Exec(`
	ALTER TABLE host_mdm_managed_certificates
	ADD COLUMN serial varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("failed to add serial columns to host_mdm_managed_certificates table: %s", err)
	}
	return nil
}

func Down_20250421085116(tx *sql.Tx) error {
	return nil
}
