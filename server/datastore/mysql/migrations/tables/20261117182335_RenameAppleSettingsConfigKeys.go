package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration("Rename macOS settings to Apple settings and custom settings to configuration profiles", Up_20261117182335, Down_20261117182335)
}

// A parent is listed before the keys under it, so the children are found at the renamed path.
var appleSettingsConfigKeys = []struct{ Parent, Old, New string }{
	{Parent: "mdm", Old: "macos_settings", New: "apple_settings"},
	{Parent: "mdm.apple_settings", Old: "custom_settings", New: "configuration_profiles"},
	{Parent: "mdm.windows_settings", Old: "custom_settings", New: "configuration_profiles"},
	{Parent: "mdm.android_settings", Old: "custom_settings", New: "configuration_profiles"},
}

func Up_20261117182335(tx *sql.Tx) error {
	for _, r := range appleSettingsConfigKeys {
		oldPath := fmt.Sprintf("$.%s.%s", r.Parent, r.Old)
		newPath := fmt.Sprintf("$.%s.%s", r.Parent, r.New)
		for _, target := range []struct{ table, column string }{
			{"app_config_json", "json_value"},
			{"teams", "config"},
			{"default_team_config_json", "json_value"},
		} {
			//nolint:gosec // built from appleSettingsConfigKeys, no user input
			stmt := fmt.Sprintf(
				`UPDATE %[1]s SET %[2]s = JSON_REMOVE(JSON_SET(%[2]s, '%[4]s', JSON_EXTRACT(%[2]s, '%[3]s')), '%[3]s')
				WHERE JSON_CONTAINS_PATH(%[2]s, 'one', '%[3]s')`,
				target.table, target.column, oldPath, newPath)
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("rename %s in %s: %w", oldPath, target.table, err)
			}
		}
	}
	return nil
}

func Down_20261117182335(tx *sql.Tx) error {
	return nil
}
