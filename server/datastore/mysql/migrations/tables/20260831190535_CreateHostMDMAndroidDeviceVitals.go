package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831190535, Down_20260831190535)
}

// Up_20260831190535 creates host_mdm_android_device_vitals, which holds the
// additional Android host vitals collected from AMAPI status reports (see
// #49791). The table is keyed by host_uuid, has no FK, and is registered in
// additionalHostRefsByUUID for host-deletion cleanup.
//
// Every vital column defaults to NULL because vitals arrive asynchronously
// via Pub/Sub and a device may not report a given section at all, depending
// on its Android version, ownership type, and the status reporting settings
// of the applied policy.
func Up_20260831190535(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE host_mdm_android_device_vitals (
  host_uuid                varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  adb_enabled              tinyint(1) DEFAULT NULL,
  passcode_protected       tinyint(1) DEFAULT NULL,
  play_protect_enabled     tinyint(1) DEFAULT NULL,
  encryption_type          varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  manufacturer             varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  security_update_version  varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  device_kernel_version    varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  bootloader_version       varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  system_update_status     varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  security_posture         varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  api_level                bigint DEFAULT NULL,
  security_posture_details json DEFAULT NULL,
  telephony_infos          json DEFAULT NULL,
  created_at               datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at               datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating host_mdm_android_device_vitals table: %w", err)
	}
	return nil
}

func Down_20260831190535(tx *sql.Tx) error {
	return nil
}
