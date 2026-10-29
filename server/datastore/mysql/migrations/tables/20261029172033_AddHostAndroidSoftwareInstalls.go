package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261029172033, Down_20261029172033)
}

func Up_20261029172033(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_android_software_installs (
  host_id           INT UNSIGNED NOT NULL,
  adam_id           VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  vpp_app_team_id   INT UNSIGNED DEFAULT NULL,
  install_type      VARCHAR(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  status            VARCHAR(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  needs_resend      TINYINT(1) NOT NULL DEFAULT '0',
  in_policy_version INT DEFAULT NULL,
  policy_built_at   DATETIME(6) NOT NULL,
  created_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_id, adam_id),
  KEY idx_host_android_software_installs_vpp_app_team_id (vpp_app_team_id),
  CONSTRAINT fk_host_android_software_installs_vpp_app_team_id FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_android_software_installs table: %w", err)
	}

	// Insert a row for an app that's never in scope for every existing Android host, so the Android profile reconciler
	// sends each host its whole app list once and deletes the row
	_, err = tx.Exec(`
INSERT IGNORE INTO host_android_software_installs (host_id, adam_id, install_type, status, policy_built_at)
SELECT host_id, 'TEMPORARY_FOR_MIGRATION', 'AVAILABLE', 'pending', NOW(6) FROM android_devices`)
	if err != nil {
		return fmt.Errorf("insert host_android_software_installs rows for existing android hosts: %w", err)
	}
	return nil
}

func Down_20261029172033(tx *sql.Tx) error {
	return nil
}
