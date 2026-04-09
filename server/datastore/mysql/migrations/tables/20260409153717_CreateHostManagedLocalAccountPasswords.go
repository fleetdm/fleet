package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260409153717, Down_20260409153717)
}

func Up_20260409153717(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE host_managed_local_account_passwords (
			host_uuid          VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			encrypted_password BLOB NOT NULL,
			command_uuid       VARCHAR(127) COLLATE utf8mb4_unicode_ci NOT NULL,
			status             VARCHAR(20)  COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			created_at         TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at         TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_uuid),
			KEY idx_hmlap_command_uuid (command_uuid),
			CONSTRAINT fk_hmlap_status FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating host_managed_local_account_passwords table: %w", err)
	}
	return nil
}

func Down_20260409153717(tx *sql.Tx) error {
	return nil
}
