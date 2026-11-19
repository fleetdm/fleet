package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261105132801, Down_20261105132801)
}

// platformPrefixConfigKeys are the stored settings renamed to put the platform first. Parent is the dot path of the object
// holding the key, the same in app_config_json and teams.config.
var platformPrefixConfigKeys = []struct{ Parent, Old, New string }{
	{Parent: "mdm", Old: "enable_recovery_lock_password", New: "macos_recovery_lock_password_enabled"},
	{Parent: "mdm.macos_setup", Old: "require_all_software_macos", New: "macos_require_all_software"},
	{Parent: "mdm.macos_setup", Old: "require_all_software_windows", New: "windows_require_all_software"},
}

// Up_20261105132801 moves platformPrefixConfigKeys to their new names in the global config and every team config, so
// upgraded servers keep them.
func Up_20261105132801(tx *sql.Tx) error {
	for _, r := range platformPrefixConfigKeys {
		oldPath := fmt.Sprintf("$.%s.%s", r.Parent, r.Old)
		newPath := fmt.Sprintf("$.%s.%s", r.Parent, r.New)
		for _, target := range []struct{ table, column string }{
			{"app_config_json", "json_value"},
			{"teams", "config"},
		} {
			//nolint:gosec // built from platformPrefixConfigKeys, no user input
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

func Down_20261105132801(tx *sql.Tx) error {
	return nil
}
