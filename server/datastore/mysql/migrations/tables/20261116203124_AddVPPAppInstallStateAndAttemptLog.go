package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261116203124, Down_20261116203124)
}

// VPPAppInstallStateQuery computes every host_vpp_app_installs row from host_vpp_software_installs. Each pointer is the
// request id one status read selects within a (host, app, platform, version bucket). Datastore tests use it as the
// oracle for the Go refresh, so the two must agree.
// Joining snapshots by pointer avoids datetime aggregate coercion to zero dates during INSERT ... SELECT.
const VPPAppInstallStateQuery = `
SELECT
	pointers.*,
	visible.created_at AS visible_created_at,
	visible.command_uuid AS visible_command_uuid,
	visible.verification_at AS visible_verification_at,
	visible.verification_failed_at AS visible_verification_failed_at,
	latest.created_at AS latest_created_at,
	latest.command_uuid AS latest_command_uuid,
	latest.verification_at AS latest_verification_at,
	latest.verification_failed_at AS latest_verification_failed_at
FROM (
	SELECT
		host_id,
		adam_id,
		platform,
		version_bucket,
		NULLIF(version_bucket, 0) AS vpp_app_team_id,
		MAX(IF(active AND verification_at IS NOT NULL, id, NULL)) AS installed_install_id,
		MAX(IF(active AND verification_at IS NULL AND verification_failed_at IS NOT NULL, id, NULL)) AS failed_install_id,
		MAX(IF(active AND verification_at IS NULL AND verification_failed_at IS NULL, id, NULL)) AS unfinished_install_id,
		MAX(IF(active AND visible_rank = 1, id, NULL)) AS visible_install_id,
		MAX(IF(canceled = 0 AND latest_rank = 1, id, NULL)) AS latest_install_id,
		MAX(IF(canceled = 0 AND verification_failed_at IS NULL AND unfailed_rank = 1, id, NULL)) AS unfailed_install_id
	FROM (
		SELECT
			id, host_id, adam_id, platform, canceled, verification_at, verification_failed_at,
			COALESCE(vpp_app_team_id, 0) AS version_bucket,
			canceled = 0 AND removed = 0 AS active,
			ROW_NUMBER() OVER (
				PARTITION BY host_id, adam_id, platform, COALESCE(vpp_app_team_id, 0), canceled = 0 AND removed = 0
				ORDER BY created_at DESC, id DESC
			) AS visible_rank,
			ROW_NUMBER() OVER (
				PARTITION BY host_id, adam_id, platform, COALESCE(vpp_app_team_id, 0), canceled = 0
				ORDER BY created_at DESC, id DESC
			) AS latest_rank,
			ROW_NUMBER() OVER (
				PARTITION BY host_id, adam_id, platform, COALESCE(vpp_app_team_id, 0), canceled = 0 AND verification_failed_at IS NULL
				ORDER BY created_at DESC, id DESC
			) AS unfailed_rank
		FROM host_vpp_software_installs
	) ranked
	GROUP BY host_id, adam_id, platform, version_bucket
) pointers
LEFT JOIN host_vpp_software_installs visible ON visible.id = pointers.visible_install_id
LEFT JOIN host_vpp_software_installs latest ON latest.id = pointers.latest_install_id`

func Up_20261116203124(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS log_vpp_app_install_attempts (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  install_id int unsigned NOT NULL,
  host_id int unsigned NOT NULL,
  adam_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  platform varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  vpp_app_team_id int unsigned DEFAULT NULL,
  command_uuid varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL,
  associated_event_id varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  attempt tinyint unsigned NOT NULL,
  outcome enum('installed','failed','retried','canceled') NOT NULL,
  mdm_status varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  user_id int unsigned DEFAULT NULL,
  policy_id int unsigned DEFAULT NULL,
  self_service tinyint(1) NOT NULL DEFAULT 0,
  requested_at timestamp NULL DEFAULT NULL,
  finished_at datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_lvaia_event (command_uuid, outcome, associated_event_id),
  KEY idx_lvaia_host_app (host_id, adam_id, platform, id),
  KEY idx_lvaia_finished (finished_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create log_vpp_app_install_attempts: %w", err)
	}

	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_vpp_app_installs (
  host_id int unsigned NOT NULL,
  adam_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  platform varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  version_bucket int unsigned NOT NULL,
  vpp_app_team_id int unsigned DEFAULT NULL,
  installed_install_id int unsigned DEFAULT NULL,
  failed_install_id int unsigned DEFAULT NULL,
  unfinished_install_id int unsigned DEFAULT NULL,
  visible_install_id int unsigned DEFAULT NULL,
  latest_install_id int unsigned DEFAULT NULL,
  unfailed_install_id int unsigned DEFAULT NULL,
  visible_created_at timestamp NULL DEFAULT NULL,
  visible_command_uuid varchar(127) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  visible_verification_at datetime(6) DEFAULT NULL,
  visible_verification_failed_at datetime(6) DEFAULT NULL,
  latest_created_at timestamp NULL DEFAULT NULL,
  latest_command_uuid varchar(127) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  latest_verification_at datetime(6) DEFAULT NULL,
  latest_verification_failed_at datetime(6) DEFAULT NULL,
  updated_at datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_id, adam_id, platform, version_bucket),
  KEY idx_hvai_version_host (vpp_app_team_id, host_id),
  KEY idx_hvai_app_host (adam_id, platform, host_id),
  CONSTRAINT fk_hvai_app FOREIGN KEY (adam_id, platform) REFERENCES vpp_apps (adam_id, platform) ON DELETE CASCADE,
  CONSTRAINT fk_hvai_version FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create host_vpp_app_installs: %w", err)
	}

	// shortcut: runs in the migration's single transaction over all history, fine at dev/QA scale; production
	// backfill is the resumable keyset job (rollout step 2).
	if _, err := tx.Exec(`
INSERT INTO host_vpp_app_installs (
	host_id, adam_id, platform, version_bucket, vpp_app_team_id,
	installed_install_id, failed_install_id, unfinished_install_id,
	visible_install_id, latest_install_id, unfailed_install_id,
	visible_created_at, visible_command_uuid, visible_verification_at, visible_verification_failed_at,
	latest_created_at, latest_command_uuid, latest_verification_at, latest_verification_failed_at
)` + VPPAppInstallStateQuery + `
ON DUPLICATE KEY UPDATE
	vpp_app_team_id = VALUES(vpp_app_team_id),
	installed_install_id = VALUES(installed_install_id),
	failed_install_id = VALUES(failed_install_id),
	unfinished_install_id = VALUES(unfinished_install_id),
	visible_install_id = VALUES(visible_install_id),
	latest_install_id = VALUES(latest_install_id),
	unfailed_install_id = VALUES(unfailed_install_id),
	visible_created_at = VALUES(visible_created_at),
	visible_command_uuid = VALUES(visible_command_uuid),
	visible_verification_at = VALUES(visible_verification_at),
	visible_verification_failed_at = VALUES(visible_verification_failed_at),
	latest_created_at = VALUES(latest_created_at),
	latest_command_uuid = VALUES(latest_command_uuid),
	latest_verification_at = VALUES(latest_verification_at),
	latest_verification_failed_at = VALUES(latest_verification_failed_at)`); err != nil {
		return fmt.Errorf("backfill host_vpp_app_installs: %w", err)
	}

	return nil
}

func Down_20261116203124(tx *sql.Tx) error {
	return nil
}
