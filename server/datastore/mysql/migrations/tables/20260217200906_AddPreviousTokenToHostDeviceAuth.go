package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260217200906, Down_20260217200906)
}

func Up_20260217200906(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_device_auth ADD COLUMN previous_token VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
		ADD INDEX idx_host_device_auth_previous_token (previous_token)`)
	if err != nil {
		return err
	}
	return nil
}

func Down_20260217200906(tx *sql.Tx) error {
	return nil
}
