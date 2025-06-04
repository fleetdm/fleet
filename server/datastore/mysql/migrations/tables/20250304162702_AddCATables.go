package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250304162702, Down_20250304162702)
}

func Up_20250304162702(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS ca_config_assets (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		type enum('digicert','custom_scep_proxy') NOT NULL,
		name VARCHAR(255) NOT NULL,
		value BLOB NOT NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
	    UNIQUE KEY idx_ca_config_assets_name (name)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create ca_config_assets table: %s", err)
	}

	_, err = tx.Exec(`
	ALTER TABLE host_mdm_managed_certificates
	ADD COLUMN not_valid_after DATETIME(6) NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to add not_valid_after column to host_mdm_managed_certificates table: %s", err)
	}
	return nil
}

func Down_20250304162702(_ *sql.Tx) error {
	return nil
}
