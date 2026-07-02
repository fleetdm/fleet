package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260701173047, Down_20260701173047)
}

func Up_20260701173047(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE mdm_apple_psso_devices (
			host_uuid  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_uuid)
		)
	`); err != nil {
		return fmt.Errorf("creating mdm_apple_psso_devices table: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE mdm_apple_psso_keys (
			kid        VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			host_uuid  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			key_type   ENUM('signing','encryption') COLLATE utf8mb4_unicode_ci NOT NULL,
			pem        TEXT COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (kid),
			CONSTRAINT fk_mdm_apple_psso_keys_host_uuid FOREIGN KEY (host_uuid) REFERENCES mdm_apple_psso_devices (host_uuid) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating mdm_apple_psso_keys table: %w", err)
	}

	return nil
}

func Down_20260701173047(tx *sql.Tx) error {
	return nil
}
