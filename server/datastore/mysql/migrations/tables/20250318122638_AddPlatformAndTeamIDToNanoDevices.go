package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250318122638, Down_20250318122638)
}

func Up_20250318122638(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE nano_devices
	ADD COLUMN platform VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	ADD COLUMN enroll_team_id INT UNSIGNED DEFAULT NULL,
	ADD CONSTRAINT fk_nano_devices_team_id FOREIGN KEY (enroll_team_id) REFERENCES teams (id) ON DELETE SET NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter nano_devices: %w", err)
	}

	// TODO(mna): parse the authenticate field of existing nano_devices that
	// don't have a corresponding host entry (deleted hosts) and set the platform
	// value to ios/ipados so that they can be recreated when they checkin.
	return nil
}

func Down_20250318122638(tx *sql.Tx) error {
	return nil
}
