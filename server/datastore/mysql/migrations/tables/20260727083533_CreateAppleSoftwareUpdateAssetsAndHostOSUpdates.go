package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260727083533, Down_20260727083533)
}

func Up_20260727083533(tx *sql.Tx) error {
	// apple_software_update_assets caches the set of available OS update
	// versions Apple's GDMF service reports for macOS/iOS, refreshed by a
	// periodic cron. first_seen_at is only set on insert; updated_at advances
	// on every successful fetch even when the version set is unchanged.
	_, err := tx.Exec(`
CREATE TABLE apple_software_update_assets (
  id                INT UNSIGNED NOT NULL AUTO_INCREMENT,
  class             enum('macos','ios') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  product_version   varchar(50)  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  build             varchar(50)  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  posting_date      date DEFAULT NULL,
  expiration_date   date DEFAULT NULL,
  supported_devices json NOT NULL,
  first_seen_at     datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  created_at        datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_asset_class_version_build (class, product_version, build)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating apple_software_update_assets table: %w", err)
	}

	// host_mdm_apple_os_updates tracks, per host, the resolved target OS
	// version/deadline when automatic enforcement is set to "latest".
	// resolved_at is set once the target has been recomputed for the host's
	// current team/setting; target_deadline is nullable until resolved.
	_, err = tx.Exec(`
CREATE TABLE host_mdm_apple_os_updates (
  host_uuid                 varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  software_update_device_id varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  target_os_version         varchar(50)  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  target_deadline           datetime(6) DEFAULT NULL,
  resolved_at               datetime(6) DEFAULT NULL,
  created_at                datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at                datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating host_mdm_apple_os_updates table: %w", err)
	}

	return nil
}

func Down_20260727083533(tx *sql.Tx) error {
	return nil
}
