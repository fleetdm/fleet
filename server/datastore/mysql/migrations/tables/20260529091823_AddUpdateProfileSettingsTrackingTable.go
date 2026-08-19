package tables

import (
	"database/sql"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260529091823, Down_20260529091823)
}

func Up_20260529091823(tx *sql.Tx) error {
	// Create the table
	_, err := tx.Exec(`
	CREATE TABLE mdm_configuration_profile_update_settings (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  windows_profile_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  apple_declaration_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY idx_mdm_config_profile_update_settings_apple_decl (apple_declaration_uuid),
  UNIQUE KEY idx_mdm_config_profile_update_settings_windows_profile (windows_profile_uuid),
  CONSTRAINT fk_mdm_config_profile_update_settings_apple_decl_uuid FOREIGN KEY (apple_declaration_uuid) REFERENCES mdm_apple_declarations (declaration_uuid) ON DELETE CASCADE,
  CONSTRAINT fk_mdm_config_profile_update_settings_windows_profile_uuid FOREIGN KEY (windows_profile_uuid) REFERENCES mdm_windows_configuration_profiles (profile_uuid) ON DELETE CASCADE,
  CONSTRAINT ck_mdm_config_profile_update_settings_exactly_one CHECK ((((IF((apple_declaration_uuid IS NULL),0,1) + IF((windows_profile_uuid IS NULL),0,1))) = 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return err
	}

	// Backfill data
	// First we check all declarations
	var declarations []struct {
		DeclarationUUID string `db:"declaration_uuid"`
		RawJSON         string `db:"raw_json"`
	}
	// We can safely ignore the names, since it's not allowed and it's ours. We only care about user uploaded declarations.
	rows, err := tx.Query("SELECT declaration_uuid, raw_json FROM mdm_apple_declarations WHERE name != 'Fleet macOS OS Updates' AND name != 'Fleet iOS OS Updates' AND name != 'Fleet iPadOS OS Updates';") //nolint:rowserrcheck // Checked inside sqlx.StructScan
	if err != nil {
		return err
	}
	if err := sqlx.StructScan(rows, &declarations); err != nil {
		return err
	}

	for _, decl := range declarations {
		// Skip all non software update enforcement declarations
		if !strings.Contains(decl.RawJSON, "com.apple.configuration.softwareupdate.enforcement.specific") {
			continue
		}

		if _, err := tx.Exec(`INSERT INTO mdm_configuration_profile_update_settings (apple_declaration_uuid) VALUES (?)`, decl.DeclarationUUID); err != nil {
			return err
		}
	}

	// Then backfill windows profiles
	var windowsProfiles []struct {
		ProfileUUID string `db:"profile_uuid"`
		SyncML      string `db:"syncml"`
	}
	rows, err = tx.Query("SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE name != 'Windows OS Updates';") //nolint:rowserrcheck // Checked inside sqlx.StructScan
	if err != nil {
		return err
	}
	if err := sqlx.StructScan(rows, &windowsProfiles); err != nil {
		return err
	}

	for _, profile := range windowsProfiles {
		// Skip all non software update profiles
		if !strings.Contains(profile.SyncML, syncml.FleetOSUpdateTargetLocURI) {
			continue
		}

		if _, err := tx.Exec(`INSERT INTO mdm_configuration_profile_update_settings (windows_profile_uuid) VALUES (?)`, profile.ProfileUUID); err != nil {
			return err
		}
	}

	return nil
}

func Down_20260529091823(tx *sql.Tx) error {
	return nil
}
