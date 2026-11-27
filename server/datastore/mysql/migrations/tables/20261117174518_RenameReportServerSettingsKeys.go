package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration("Rename report server settings", Up_20261117174518, Down_20261117174518)
}

// reportServerSettingsKeyRenames are the stored server settings renamed from "query" to "report".
var reportServerSettingsKeyRenames = []struct{ Old, New string }{
	{Old: "live_query_disabled", New: "live_reporting_disabled"},
	{Old: "query_reports_disabled", New: "discard_reports_data"},
	{Old: "query_report_cap", New: "report_cap"},
}

// Up_20261117174518 moves the stored server settings to their new names, so an upgraded server keeps them. They only
// exist in the global config.
func Up_20261117174518(tx *sql.Tx) error {
	for _, r := range reportServerSettingsKeyRenames {
		oldPath := "$.server_settings." + r.Old
		newPath := "$.server_settings." + r.New
		//nolint:gosec // built from reportServerSettingsKeyRenames, no user input
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

func Down_20261117174518(tx *sql.Tx) error {
	return nil
}
