package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250422095806, Down_20250422095806)
}

func Up_20250422095806(tx *sql.Tx) error {
	// fleet_variables stores variable names that can be used in configuration
	// profiles, scripts, etc. and that get replaced server-side with a
	// fleet-known value before being used.
	//
	// Note that at the time this migration was created, fleet _secrets_ are not
	// stored in this table.
	// See https://github.com/fleetdm/fleet/issues/28035#issuecomment-2810400682
	// for more details.

	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS fleet_variables (
		id          INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name        VARCHAR(255) NOT NULL DEFAULT '',
		is_prefix   TINYINT(1) NOT NULL DEFAULT 0,
		created_at  DATETIME(6) NOT NULL DEFAULT NOW(6),

		UNIQUE KEY idx_fleet_variables_name_is_prefix (name, is_prefix)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`)

	if err != nil {
		return fmt.Errorf("failed to create fleet_variables table: %s", err)
	}

	// to ensure the tuples (apple profile, fleet variable) and (windows profile,
	// fleet variable) are unique and have proper foreign keys, we need two
	// distinct columns for the two types of profiles, and we use unique keys,
	// foreign keys and check constraints to enforce that this is the case.
	// Same approach as for mdm_configuration_profile_labels.
	//
	// (Note that we don't support fleet variables in Windows profiles currently,
	// but since this pattern is somewhat complex, I opted to create the table
	// with Windows support immediately to avoid potential issues in a future
	// ALTER TABLE).
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS mdm_configuration_profile_variables (
		id                   INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		apple_profile_uuid   VARCHAR(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
		windows_profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
		fleet_variable_id    INT UNSIGNED NOT NULL,
		created_at           DATETIME(6) NOT NULL DEFAULT NOW(6),

		UNIQUE KEY idx_mdm_configuration_profile_variables_apple_variable (apple_profile_uuid, fleet_variable_id),
		UNIQUE KEY idx_mdm_configuration_profile_variables_windows_label_name (windows_profile_uuid, fleet_variable_id),
		CONSTRAINT fk_mdm_configuration_profile_variables_apple_profile_uuid
			FOREIGN KEY (apple_profile_uuid) REFERENCES mdm_apple_configuration_profiles (profile_uuid) ON DELETE CASCADE,
		CONSTRAINT fk_mdm_configuration_profile_variables_windows_profile_uuid
			FOREIGN KEY (windows_profile_uuid) REFERENCES mdm_windows_configuration_profiles (profile_uuid) ON DELETE CASCADE,
		CONSTRAINT mdm_configuration_profile_variables_fleet_variable_id 
			FOREIGN KEY (fleet_variable_id) REFERENCES fleet_variables (id) ON DELETE CASCADE,
		CONSTRAINT ck_mdm_configuration_profile_variables_apple_or_windows 
			CHECK (((apple_profile_uuid is null) <> (windows_profile_uuid is null)))
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_configuration_profile_variables table: %s", err)
	}

	// TODO(mna): insert known Fleet variables, scan profile contents to insert the relationships.
	return nil
}

func Down_20250422095806(tx *sql.Tx) error {
	return nil
}
