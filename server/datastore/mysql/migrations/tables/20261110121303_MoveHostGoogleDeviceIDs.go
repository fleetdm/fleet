package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261110121303, Down_20261110121303)
}

func Up_20261110121303(tx *sql.Tx) error {
	// The collation matches host_google_device_ids, so IDs that were unique there
	// stay unique here.
	if !columnExists(tx, "google_conditional_access_devices", "google_device_id") {
		if _, err := tx.Exec(`
			ALTER TABLE google_conditional_access_devices
				ADD COLUMN google_device_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NULL AFTER customer_id,
				ADD UNIQUE KEY idx_google_conditional_access_devices_google_device_id (google_device_id)
		`); err != nil {
			return fmt.Errorf("add google_device_id to google_conditional_access_devices: %w", err)
		}
	}

	if !tableExists(tx, "host_google_device_ids") {
		return nil
	}

	// A copied row has no customer and no DeviceUsers: the sync cron fills them in once
	// the integration is connected.
	if _, err := tx.Exec(`
		INSERT INTO google_conditional_access_devices (host_uuid, customer_id, google_device_id)
		SELECT h.uuid, '', g.google_device_id
		FROM host_google_device_ids g
		JOIN hosts h ON h.id = g.host_id
		WHERE h.uuid <> ''
		ON DUPLICATE KEY UPDATE
			google_device_id = COALESCE(google_conditional_access_devices.google_device_id, VALUES(google_device_id))
	`); err != nil {
		return fmt.Errorf("copy host_google_device_ids to google_conditional_access_devices: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE IF EXISTS host_google_device_ids`); err != nil {
		return fmt.Errorf("drop host_google_device_ids: %w", err)
	}
	return nil
}

func Down_20261110121303(tx *sql.Tx) error {
	return nil
}
