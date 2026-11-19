package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261105133221, Down_20261105133221)
}

// windowsMDMConfigKeyRenames are the Windows MDM settings renamed to start with the platform and include "mdm".
var windowsMDMConfigKeyRenames = []struct{ Parent, Old, New string }{
	{Parent: "mdm", Old: "windows_enabled_and_configured", New: "windows_mdm_enabled"},
	{Parent: "mdm", Old: "windows_migration_enabled", New: "windows_mdm_migration_enabled"},
}

// Up_20261105133221 moves the stored Windows MDM settings to their new names, so an upgraded server keeps Windows MDM
// on. They only exist in the global config.
func Up_20261105133221(tx *sql.Tx) error {
	for _, r := range windowsMDMConfigKeyRenames {
		oldPath := fmt.Sprintf("$.%s.%s", r.Parent, r.Old)
		newPath := fmt.Sprintf("$.%s.%s", r.Parent, r.New)
		//nolint:gosec // built from windowsMDMConfigKeyRenames, no user input
		stmt := fmt.Sprintf(
			`UPDATE app_config_json SET json_value = JSON_REMOVE(JSON_SET(json_value, '%[2]s', JSON_EXTRACT(json_value, '%[1]s')), '%[1]s')
			WHERE JSON_CONTAINS_PATH(json_value, 'one', '%[1]s')`,
			oldPath, newPath)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("rename %s in app_config_json: %w", oldPath, err)
		}
	}
	return nil
}

func Down_20261105133221(tx *sql.Tx) error {
	return nil
}
