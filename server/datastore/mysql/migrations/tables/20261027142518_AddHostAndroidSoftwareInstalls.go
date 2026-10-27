package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261027142518, Down_20261027142518)
}

func Up_20261027142518(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_android_software_installs (
  id                INT UNSIGNED NOT NULL AUTO_INCREMENT,
  host_id           INT UNSIGNED NOT NULL,
  adam_id           VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  vpp_app_team_id   INT UNSIGNED DEFAULT NULL,
  status            VARCHAR(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  in_policy_version INT DEFAULT NULL,
  needs_resend      TINYINT(1) NOT NULL DEFAULT '0',
  policy_built_at   DATETIME(6) NOT NULL,
  created_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_host_android_software_installs_host_adam (host_id, adam_id),
  KEY idx_host_android_software_installs_vpp_app_team_id (vpp_app_team_id),
  CONSTRAINT fk_host_android_software_installs_vpp_app_team_id FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_android_software_installs table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS log_android_software_installs (
  id                INT UNSIGNED NOT NULL AUTO_INCREMENT,
  host_id           INT UNSIGNED NOT NULL,
  adam_id           VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  vpp_app_team_id   INT UNSIGNED DEFAULT NULL,
  status            VARCHAR(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  in_policy_version INT DEFAULT NULL,
  created_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_log_android_software_installs_host_adam_status (host_id, adam_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create log_android_software_installs table: %w", err)
	}

	return nil
}

func Down_20261027142518(tx *sql.Tx) error {
	return nil
}
