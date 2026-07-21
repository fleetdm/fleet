package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260713115453, Down_20260713115453)
}

func Up_20260713115453(tx *sql.Tx) error {
	// host_mdm_apple_device_names tracks the enforcement state of the host-name
	// template for Apple hosts (macOS, iOS, iPadOS). A NULL status means the row
	// is queued for the cron to pick up and send a Settings/DeviceName command.
	// expected_device_name is nullable because rows are
	// created before the cron resolves the template into a concrete name.
	_, err := tx.Exec(`
CREATE TABLE host_mdm_apple_device_names (
  host_uuid            varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  status               varchar(20)  COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  command_uuid         varchar(127) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  expected_device_name varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  detail               text COLLATE utf8mb4_unicode_ci,
  created_at           datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at           datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_uuid),
  KEY idx_host_mdm_apple_device_names_status (status),
  KEY idx_host_mdm_apple_device_names_command_uuid (command_uuid),
  CONSTRAINT host_mdm_apple_device_names_status FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating host_mdm_apple_device_names table: %w", err)
	}
	return nil
}

func Down_20260713115453(tx *sql.Tx) error {
	return nil
}
