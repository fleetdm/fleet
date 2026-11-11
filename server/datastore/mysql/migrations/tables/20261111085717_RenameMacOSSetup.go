package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261111085717, Down_20261111085717)
}

// macOSSetupRenames are the stored setup experience settings renamed for Fleet 5. Parent is the dot path of the object
// holding the key, the same in every config table. The parent is renamed first, so the nested keys are found under
// setup_experience.
var macOSSetupRenames = []struct{ Parent, Old, New string }{
	{Parent: "mdm", Old: "macos_setup", New: "setup_experience"},

	{Parent: "mdm.setup_experience", Old: "bootstrap_package", New: "macos_bootstrap_package"},
	{Parent: "mdm.setup_experience", Old: "macos_setup_assistant", New: "apple_setup_assistant"},
	{Parent: "mdm.setup_experience", Old: "enable_release_device_manually", New: "apple_enable_release_device_manually"},
	{Parent: "mdm.setup_experience", Old: "manual_agent_install", New: "macos_manual_agent_install"},
	{Parent: "mdm.setup_experience", Old: "script", New: "macos_script"},
}

// Up_20261111085717 moves macOSSetupRenames to their new names in the global config, every team config, and the
// default team config, so upgraded servers keep their setup experience settings. Without it the server would read
// defaults and silently turn off end user authentication. Fresh installs need it too: the baseline seeds macos_setup.
func Up_20261111085717(tx *sql.Tx) error {
	for _, r := range macOSSetupRenames {
		oldPath := fmt.Sprintf("$.%s.%s", r.Parent, r.Old)
		newPath := fmt.Sprintf("$.%s.%s", r.Parent, r.New)
		for _, target := range []struct{ table, column string }{
			{"app_config_json", "json_value"},
			{"teams", "config"},
			{"default_team_config_json", "json_value"},
		} {
			//nolint:gosec // built from macOSSetupRenames, no user input
			stmt := fmt.Sprintf(
				`UPDATE %[1]s SET %[2]s = JSON_REMOVE(JSON_SET(%[2]s, '%[4]s', JSON_EXTRACT(%[2]s, '%[3]s')), '%[3]s')
				WHERE JSON_CONTAINS_PATH(%[2]s, 'one', '%[3]s')`,
				target.table, target.column, oldPath, newPath,
			)
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("rename %s in %s: %w", oldPath, target.table, err)
			}
		}
	}
	return nil
}

func Down_20261111085717(tx *sql.Tx) error {
	return nil
}
