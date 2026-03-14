package tables

import (
	"database/sql"
	"fmt"
	"time"
)

func init() {
	MigrationClient.AddMigration(Up_20260314000000, Down_20260314000000)
}

func Up_20260314000000(tx *sql.Tx) error {
	// scim_user_custom_attributes stores custom IdP attributes for SCIM users.
	// These are key-value pairs that come from the SCIM Enterprise User extension
	// or other custom schema extensions from the IdP (e.g., costCenter, manager).
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS scim_user_custom_attributes (
		id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		scim_user_id INT UNSIGNED NOT NULL,
		name        VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		value       VARCHAR(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
		created_at  DATETIME(6) NOT NULL DEFAULT NOW(6),
		updated_at  DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

		UNIQUE KEY idx_scim_user_custom_attributes_user_name (scim_user_id, name),
		CONSTRAINT fk_scim_user_custom_attributes_scim_user_id
			FOREIGN KEY (scim_user_id) REFERENCES scim_users(id) ON DELETE CASCADE
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("failed to create scim_user_custom_attributes table: %w", err)
	}

	// Add a prefix fleet variable for custom IdP attributes so they can be used
	// in MDM configuration profiles as $FLEET_VAR_HOST_END_USER_IDP_CUSTOM_<ATTRIBUTE_NAME>
	createdAt := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
	_, err = tx.Exec(
		"INSERT INTO fleet_variables (name, is_prefix, created_at) VALUES ('FLEET_VAR_HOST_END_USER_IDP_CUSTOM_', 1, ?)",
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert FLEET_VAR_HOST_END_USER_IDP_CUSTOM_ into fleet_variables: %w", err)
	}

	// Add manager column to scim_users (stores the displayName of the user's manager)
	_, err = tx.Exec(`ALTER TABLE scim_users ADD COLUMN manager VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER department`)
	if err != nil {
		return fmt.Errorf("failed to add manager column to scim_users: %w", err)
	}

	// Add fleet variable for the manager attribute so it can be used in
	// MDM configuration profiles as $FLEET_VAR_HOST_END_USER_IDP_MANAGER
	_, err = tx.Exec(
		"INSERT INTO fleet_variables (name, is_prefix, created_at) VALUES ('FLEET_VAR_HOST_END_USER_IDP_MANAGER', 0, ?)",
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert FLEET_VAR_HOST_END_USER_IDP_MANAGER into fleet_variables: %w", err)
	}

	return nil
}

func Down_20260314000000(tx *sql.Tx) error {
	return nil
}
