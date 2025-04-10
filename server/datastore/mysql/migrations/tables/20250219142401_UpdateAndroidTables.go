package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250219142401, Down_20250219142401)
}

func Up_20250219142401(tx *sql.Tx) error {
	if !columnsExists(tx, "android_enterprises", "signup_token", "pubsub_topic_id") {
		_, err := tx.Exec(`ALTER TABLE android_enterprises
			-- Authentication token for callback endpoint to create enterprise
			ADD COLUMN signup_token VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    		-- PubSub topic_id
			ADD COLUMN pubsub_topic_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
    		`)
		if err != nil {
			return fmt.Errorf("failed to update android_enterprise table: %w", err)
		}
	}

	_, err := tx.Exec(`CREATE TABLE android_devices (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    		host_id INT UNSIGNED NOT NULL,
    		device_id VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			-- The enterprise_specific_id uniquely identifies personally-owned devices.
    		enterprise_specific_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
    		-- We could have a flow in the future where policy is assigned after enrollment.
    		android_policy_id INT UNSIGNED NULL,
    		last_policy_sync_time DATETIME(3) NULL,
    		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
    		updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
    		PRIMARY KEY (id),
    		UNIQUE KEY idx_android_devices_host_id (host_id),
    		UNIQUE KEY idx_android_devices_device_id (device_id),
    		UNIQUE KEY idx_android_devices_enterprise_specific_id (enterprise_specific_id)
   		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create android_devices table: %w", err)
	}

	return nil
}

func Down_20250219142401(_ *sql.Tx) error {
	return nil
}
