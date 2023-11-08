package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231106144110, Down_20231106144110)
}

func Up_20231106144110(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE mdm_apple_delivery_status RENAME TO mdm_delivery_status;
`)
	if err != nil {
		return fmt.Errorf("failed to rename mdm_apple_delivery_table table: %w", err)
	}
	_, err = tx.Exec(`
ALTER TABLE mdm_apple_operation_types RENAME TO mdm_operation_types;
`)
	if err != nil {
		return fmt.Errorf("failed to rename mdm_apple_operation_types table: %w", err)
	}

	_, err = tx.Exec(`
-- track the team/no-team profiles - those represent the desired state of
-- profiles per teams.
CREATE TABLE mdm_windows_configuration_profiles (
  -- this is typically called just id but this is consistent with the apple
  -- profiles table.
  profile_id   INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,

  -- this is 0 for no-team, or > 0 for a given team id, same as the apple
  -- profiles table.
  team_id      INT(10) UNSIGNED NOT NULL DEFAULT '0',

  name         VARCHAR(255) NOT NULL,
  syncml       MEDIUMBLOB NOT NULL,
  created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY (profile_id),
  UNIQUE KEY idx_mdm_windows_configuration_profiles_team_id_name (team_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_windows_configuration_profiles table: %w", err)
	}

	_, err = tx.Exec(`
-- track the current status of each profile for each host.
CREATE TABLE host_mdm_windows_profiles (
  -- this is consistent with the apple profiles table, there is no FK on the
  -- profiles because hosts may have profiles that don't exist anymore in that
  -- table.
  profile_id     INT(10) UNSIGNED NOT NULL,
  host_uuid      VARCHAR(255) NOT NULL,

  -- indicates that profile's status regarding the operation type on that host
  -- (pending, failed, etc.).
  status         VARCHAR(20) NULL,

  -- indicates whether the profile is being installed or removed.
  operation_type VARCHAR(20) NULL,

  detail         TEXT NULL,
  command_uuid   VARCHAR(127) NOT NULL,
  profile_name   VARCHAR(255) NOT NULL DEFAULT '',
  retries        TINYINT(3) UNSIGNED NOT NULL DEFAULT '0',

  PRIMARY KEY (host_uuid, profile_id),
  FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE,
  FOREIGN KEY (operation_type) REFERENCES mdm_operation_types (operation_type) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("failed to create host_mdm_windows_profiles table: %w", err)
	}

	return nil
}

func Down_20231106144110(tx *sql.Tx) error {
	return nil
}
