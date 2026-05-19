package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260518160000, Down_20260518160000)
}

func Up_20260518160000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE mdm_apple_psso_devices (
			host_id              INT UNSIGNED NOT NULL,
			device_uuid          VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			signing_key_pem      TEXT          COLLATE utf8mb4_unicode_ci NOT NULL,
			encryption_key_pem   TEXT          COLLATE utf8mb4_unicode_ci NOT NULL,
			key_exchange_key     VARBINARY(64) NOT NULL,
			created_at           TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at           TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_id),
			UNIQUE KEY idx_mdm_apple_psso_devices_device_uuid (device_uuid),
			CONSTRAINT fk_mdm_apple_psso_devices_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating mdm_apple_psso_devices table: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE mdm_apple_psso_key_ids (
			kid                  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			host_id              INT UNSIGNED NOT NULL,
			key_type             ENUM('signing','encryption') COLLATE utf8mb4_unicode_ci NOT NULL,
			pem                  TEXT          COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at           TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			PRIMARY KEY (kid),
			UNIQUE KEY idx_mdm_apple_psso_key_ids_host_type (host_id, key_type),
			CONSTRAINT fk_mdm_apple_psso_key_ids_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating mdm_apple_psso_key_ids table: %w", err)
	}

	return nil
}

func Down_20260518160000(tx *sql.Tx) error {
	return nil
}
