package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261105074613, Down_20261105074613)
}

func Up_20261105074613(tx *sql.Tx) error {
	// A single global row, pinned to id 1 so concurrent saves upsert the same row.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS google_conditional_access_integrations (
			id                  INT UNSIGNED NOT NULL DEFAULT 1,
			customer_id         VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
			admin_email         VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			proxy_server_secret VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			oauth_client_id     VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			state               VARCHAR(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
			error_message       TEXT COLLATE utf8mb4_unicode_ci NULL,
			created_at          DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
			updated_at          DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("create google_conditional_access_integrations: %w", err)
	}

	// Keyed by host UUID and not in the host deletion cascade: the row and its names
	// must outlive the host until every name is marked UNMANAGED, and a host that
	// re-enrolls with the same UUID finds its row again.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS google_conditional_access_devices (
			host_uuid                     VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			registration_token_hash       BINARY(32) NULL,
			registration_token_expires_at DATETIME(6) NULL,
			registration_attempts         INT UNSIGNED NOT NULL DEFAULT 0,
			first_registration_attempt_at DATETIME(6) NULL,
			created_at                    DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
			updated_at                    DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_uuid),
			UNIQUE KEY idx_google_conditional_access_devices_registration_token_hash (registration_token_hash)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("create google_conditional_access_devices: %w", err)
	}

	// Google resource IDs are case-sensitive, so the name uses a binary collation
	// for the uniqueness check across hosts.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS google_conditional_access_device_users (
			host_uuid        VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			device_user_name VARCHAR(255) COLLATE utf8mb4_bin NOT NULL,
			state            VARCHAR(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
			error_message    TEXT COLLATE utf8mb4_unicode_ci NULL,
			created_at       DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
			updated_at       DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (device_user_name),
			KEY idx_google_conditional_access_device_users_host_uuid (host_uuid),
			KEY idx_google_conditional_access_device_users_state (state),
			CONSTRAINT fk_google_conditional_access_device_users_host_uuid
				FOREIGN KEY (host_uuid) REFERENCES google_conditional_access_devices (host_uuid) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("create google_conditional_access_device_users: %w", err)
	}

	return nil
}

func Down_20261105074613(tx *sql.Tx) error {
	return nil
}
