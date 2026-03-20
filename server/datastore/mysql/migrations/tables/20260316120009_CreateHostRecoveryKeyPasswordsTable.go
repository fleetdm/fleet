package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260316120009, Down_20260316120009)
}

func Up_20260316120009(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE host_recovery_key_passwords (
			host_uuid varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			encrypted_password BLOB NOT NULL,
			status VARCHAR(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			operation_type VARCHAR(20) COLLATE utf8mb4_unicode_ci NOT NULL,
			error_message TEXT COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			deleted TINYINT(1) NOT NULL DEFAULT 0,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_uuid),
			KEY status (status),
			KEY operation_type (operation_type),
			KEY deleted (deleted),
			CONSTRAINT host_recovery_key_passwords_ibfk_1 FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE,
			CONSTRAINT host_recovery_key_passwords_ibfk_2 FOREIGN KEY (operation_type) REFERENCES mdm_operation_types (operation_type) ON UPDATE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating host_recovery_key_passwords table: %w", err)
	}
	return nil
}

func Down_20260316120009(tx *sql.Tx) error {
	return nil
}
