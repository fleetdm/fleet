package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250313163430, Down_20250313163430)
}

func Up_20250313163430(tx *sql.Tx) error {
	if columnsExists(tx, "host_mdm_managed_certificates", "type", "ca_name") {
		return nil
	}
	_, err := tx.Exec(`
	ALTER TABLE host_mdm_managed_certificates
	ADD COLUMN type enum('digicert', 'custom_scep_proxy', 'ndes') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ndes' AFTER profile_uuid,
	ADD COLUMN ca_name varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'NDES' AFTER type
	`)
	if err != nil {
		return fmt.Errorf("failed to add type and ca_name columns to host_mdm_managed_certificates table: %s", err)
	}

	return nil
}

func Down_20250313163430(_ *sql.Tx) error {
	return nil
}
