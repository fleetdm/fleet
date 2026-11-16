package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261116102943, Down_20261116102943)
}

// Up_20261116102943 copies the webhook or ticket choice from the failing_policies_webhook.policy_ids
// settings lists onto each policy. The lists aren't validated, so each one only applies to policies
// in its own scope; foreign or deleted IDs are ignored.
func Up_20261116102943(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE policies ADD COLUMN webhooks_and_tickets_enabled TINYINT(1) NOT NULL DEFAULT '0'`); err != nil {
		return fmt.Errorf("add policies.webhooks_and_tickets_enabled: %w", err)
	}

	for _, stmt := range []struct{ scope, query string }{
		{"global", `
			UPDATE policies p
			JOIN app_config_json ac ON ac.id = 1
			SET p.webhooks_and_tickets_enabled = 1
			WHERE p.team_id IS NULL
				AND JSON_CONTAINS(ac.json_value->'$.webhook_settings.failing_policies_webhook.policy_ids', CAST(p.id AS JSON))`},
		{"team", `
			UPDATE policies p
			JOIN teams t ON t.id = p.team_id
			SET p.webhooks_and_tickets_enabled = 1
			WHERE JSON_CONTAINS(t.config->'$.webhook_settings.failing_policies_webhook.policy_ids', CAST(p.id AS JSON))`},
		{"no team", `
			UPDATE policies p
			JOIN default_team_config_json dc ON dc.id = 1
			SET p.webhooks_and_tickets_enabled = 1
			WHERE p.team_id = 0
				AND JSON_CONTAINS(dc.json_value->'$.webhook_settings.failing_policies_webhook.policy_ids', CAST(p.id AS JSON))`},
	} {
		if _, err := tx.Exec(stmt.query); err != nil {
			return fmt.Errorf("copy %s policy_ids to policies: %w", stmt.scope, err)
		}
	}
	return nil
}

func Down_20261116102943(tx *sql.Tx) error {
	return nil
}
