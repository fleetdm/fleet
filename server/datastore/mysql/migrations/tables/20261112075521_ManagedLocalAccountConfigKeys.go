package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261112075521, Down_20261112075521)
}

// Up_20261112075521 moves the managed local account settings to their Fleet 5 names under setup_experience in the global
// config and every team config. default_team_config_json is left alone: nothing acts on its MDM settings.
func Up_20261112075521(tx *sql.Tx) error {
	for _, target := range []struct{ table, column string }{
		{"app_config_json", "json_value"},
		{"teams", "config"},
	} {
		for _, r := range []struct{ Old, New string }{
			{"enable_managed_local_account", "macos_managed_local_account_enabled"},
			{"end_user_local_account_type", "macos_end_user_local_account_type"},
		} {
			oldPath := "$.mdm.setup_experience." + r.Old
			//nolint:gosec // built from constants, no user input
			stmt := fmt.Sprintf(
				`UPDATE %[1]s SET %[2]s = JSON_REMOVE(JSON_SET(%[2]s, '$.mdm.setup_experience.%[4]s', JSON_EXTRACT(%[2]s, '%[3]s')), '%[3]s')
				WHERE JSON_CONTAINS_PATH(%[2]s, 'one', '%[3]s')`,
				target.table, target.column, oldPath, r.New)
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("rename %s in %s: %w", oldPath, target.table, err)
			}
		}

		// JSON_SET silently does nothing under a missing or JSON null parent, so setup_experience is made an object first.
		//nolint:gosec // built from constants, no user input
		stmt := fmt.Sprintf(
			`UPDATE %[1]s SET %[2]s = JSON_REMOVE(
				JSON_SET(%[2]s,
					'$.mdm.setup_experience', IF(JSON_TYPE(JSON_EXTRACT(%[2]s, '$.mdm.setup_experience')) = 'OBJECT',
						JSON_EXTRACT(%[2]s, '$.mdm.setup_experience'), JSON_OBJECT()),
					'$.mdm.setup_experience.windows_managed_local_account_enabled',
					JSON_EXTRACT(%[2]s, '$.mdm.windows_settings.enable_managed_local_account')),
				'$.mdm.windows_settings.enable_managed_local_account')
			WHERE JSON_CONTAINS_PATH(%[2]s, 'one', '$.mdm.windows_settings.enable_managed_local_account')`,
			target.table, target.column)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("move windows managed local account setting in %s: %w", target.table, err)
		}
	}
	return nil
}

func Down_20261112075521(tx *sql.Tx) error {
	return nil
}
