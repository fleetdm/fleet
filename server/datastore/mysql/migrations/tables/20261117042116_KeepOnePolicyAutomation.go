package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration("Keep at most one automation per policy", Up_20261117042116, Down_20261117042116)
}

// Up_20261117042116 leaves each policy with at most one automation, keeping the first in this order: conditional
// access, software, script, configuration profile, calendar, webhook or ticket. Calendar never worked on "All fleets"
// or "Unassigned" policies, so it's turned off there first and can't win. The failing_policies_webhook.policy_ids
// lists, already copied onto the policies, are removed from the settings.
//
// Policies keep their updated_at, so the change doesn't look like a policy edit (which would rerun membership cleanup).
func Up_20261117042116(tx *sql.Tx) error {
	const policyIDsPath = `'$.webhook_settings.failing_policies_webhook.policy_ids'`
	for _, stmt := range []struct{ desc, query string }{
		{"turn calendar off on All fleets and Unassigned policies", `
			UPDATE policies SET calendar_events_enabled = 0, updated_at = updated_at
			WHERE calendar_events_enabled = 1 AND (team_id IS NULL OR team_id = 0)`},
		{"keep conditional access", `
			UPDATE policies SET software_installer_id = NULL, vpp_apps_teams_id = NULL, script_id = NULL,
				resend_apple_profile_uuid = NULL, resend_windows_profile_uuid = NULL,
				calendar_events_enabled = 0, webhooks_and_tickets_enabled = 0, updated_at = updated_at
			WHERE conditional_access_enabled = 1`},
		{"keep software", `
			UPDATE policies SET script_id = NULL, resend_apple_profile_uuid = NULL, resend_windows_profile_uuid = NULL,
				calendar_events_enabled = 0, webhooks_and_tickets_enabled = 0, updated_at = updated_at
			WHERE software_installer_id IS NOT NULL OR vpp_apps_teams_id IS NOT NULL`},
		{"keep script", `
			UPDATE policies SET resend_apple_profile_uuid = NULL, resend_windows_profile_uuid = NULL,
				calendar_events_enabled = 0, webhooks_and_tickets_enabled = 0, updated_at = updated_at
			WHERE script_id IS NOT NULL`},
		{"keep configuration profile", `
			UPDATE policies SET calendar_events_enabled = 0, webhooks_and_tickets_enabled = 0, updated_at = updated_at
			WHERE resend_apple_profile_uuid IS NOT NULL OR resend_windows_profile_uuid IS NOT NULL`},
		{"keep calendar", `
			UPDATE policies SET webhooks_and_tickets_enabled = 0, updated_at = updated_at
			WHERE calendar_events_enabled = 1 AND webhooks_and_tickets_enabled = 1`},
		{"remove global policy_ids", `
			UPDATE app_config_json SET json_value = JSON_REMOVE(json_value, ` + policyIDsPath + `)
			WHERE JSON_CONTAINS_PATH(json_value, 'one', ` + policyIDsPath + `)`},
		{"remove fleet policy_ids", `
			UPDATE teams SET config = JSON_REMOVE(config, ` + policyIDsPath + `)
			WHERE JSON_CONTAINS_PATH(config, 'one', ` + policyIDsPath + `)`},
		{"remove Unassigned policy_ids", `
			UPDATE default_team_config_json SET json_value = JSON_REMOVE(json_value, ` + policyIDsPath + `)
			WHERE JSON_CONTAINS_PATH(json_value, 'one', ` + policyIDsPath + `)`},
	} {
		if _, err := tx.Exec(stmt.query); err != nil {
			return fmt.Errorf("%s: %w", stmt.desc, err)
		}
	}
	return nil
}

func Down_20261117042116(tx *sql.Tx) error {
	return nil
}
