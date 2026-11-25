package tables

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261117042116(t *testing.T) {
	db := applyUpToPrev(t)
	ctx := context.Background()

	type policy struct {
		name                              string
		teamID                            any
		conditionalAccess                 bool
		installerID, vppID, scriptID      any
		appleProfile, windowsProfile      any
		calendar, webhookOrTicket         bool
		patch                             bool
		wantAutomations                   string
		wantPatchTitleID, wantPatchClosed bool
	}
	policies := []policy{
		{name: "conditional access and software", teamID: 1, conditionalAccess: true, installerID: 10, wantAutomations: "conditional_access"},
		{name: "patch policy", teamID: 1, conditionalAccess: true, installerID: 11, patch: true, wantAutomations: "conditional_access",
			wantPatchTitleID: true, wantPatchClosed: true},
		{name: "app store app and script", teamID: 1, vppID: 20, scriptID: 30, wantAutomations: "software"},
		{name: "script, profile and webhook", teamID: 1, scriptID: 31, appleProfile: "a-1", webhookOrTicket: true, wantAutomations: "script"},
		{name: "profile and calendar", teamID: 1, windowsProfile: "w-1", calendar: true, wantAutomations: "configuration_profile"},
		{name: "calendar and webhook", teamID: 1, calendar: true, webhookOrTicket: true, wantAutomations: "calendar"},
		{name: "All fleets calendar and webhook", teamID: nil, calendar: true, webhookOrTicket: true, wantAutomations: "webhook_or_ticket"},
		{name: "Unassigned calendar and webhook", teamID: 0, calendar: true, webhookOrTicket: true, wantAutomations: "webhook_or_ticket"},
		{name: "Unassigned calendar", teamID: 0, calendar: true, wantAutomations: ""},
		{name: "script only", teamID: 1, scriptID: 32, wantAutomations: "script"},
		{name: "webhook only", teamID: 1, webhookOrTicket: true, wantAutomations: "webhook_or_ticket"},
		{name: "none", teamID: 1, wantAutomations: ""},
	}

	// The automations only need to be set, so their referenced rows aren't created.
	conn, err := db.Connx(ctx)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS = 0`)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `INSERT INTO teams (id, name, config) VALUES
		(1, 'A', '{"webhook_settings": {"failing_policies_webhook": {"enable_failing_policies_webhook": true, "policy_ids": [4, 6]}}}'),
		(2, 'B', NULL),
		(3, 'C', '{"features": {}}')`)
	require.NoError(t, err)
	updatedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range policies {
		typ, patchTitleID := "dynamic", any(nil)
		if p.patch {
			typ, patchTitleID = "patch", 9
		}
		_, err := conn.ExecContext(ctx, `INSERT INTO policies (name, query, description, checksum, team_id,
			conditional_access_enabled, software_installer_id, vpp_apps_teams_id, script_id,
			resend_apple_profile_uuid, resend_windows_profile_uuid, calendar_events_enabled, webhooks_and_tickets_enabled,
			type, patch_software_title_id, patch_when_closed, continuous_automations_enabled, updated_at)
			VALUES (?, 'SELECT 1', '', UNHEX(MD5(?)), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.name, p.name, p.teamID, p.conditionalAccess, p.installerID, p.vppID, p.scriptID, p.appleProfile, p.windowsProfile,
			p.calendar, p.webhookOrTicket, typ, patchTitleID, p.patch, p.patch, updatedAt)
		require.NoError(t, err)
	}
	_, err = conn.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS = 1`)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_MERGE_PATCH(json_value,
		'{"webhook_settings": {"failing_policies_webhook": {"enable_failing_policies_webhook": true, "destination_url": "https://example.com", "policy_ids": [7]}}}')
		WHERE id = 1`)
	execNoErr(t, db, `INSERT INTO default_team_config_json (id, json_value) VALUES (1, '{"webhook_settings": {"failing_policies_webhook": {"policy_ids": [8]}}}')
		ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`)

	check := func() {
		var got []struct {
			Name            string    `db:"name"`
			Automations     string    `db:"automations"`
			PatchTitleID    *uint     `db:"patch_software_title_id"`
			PatchWhenClosed bool      `db:"patch_when_closed"`
			UpdatedAt       time.Time `db:"updated_at"`
		}
		require.NoError(t, sqlx.Select(db, &got, `SELECT name, CONCAT_WS(',',
				IF(conditional_access_enabled, 'conditional_access', NULL),
				IF(software_installer_id IS NOT NULL OR vpp_apps_teams_id IS NOT NULL, 'software', NULL),
				IF(script_id IS NOT NULL, 'script', NULL),
				IF(resend_apple_profile_uuid IS NOT NULL OR resend_windows_profile_uuid IS NOT NULL, 'configuration_profile', NULL),
				IF(calendar_events_enabled, 'calendar', NULL),
				IF(webhooks_and_tickets_enabled, 'webhook_or_ticket', NULL)) AS automations,
			patch_software_title_id, patch_when_closed, updated_at
			FROM policies ORDER BY id`))
		require.Len(t, got, len(policies))
		for i, p := range policies {
			require.Equal(t, p.name, got[i].Name)
			require.Equal(t, p.wantAutomations, got[i].Automations, p.name)
			require.Equal(t, p.wantPatchTitleID, got[i].PatchTitleID != nil, p.name)
			require.Equal(t, p.wantPatchClosed, got[i].PatchWhenClosed, p.name)
			require.True(t, updatedAt.Equal(got[i].UpdatedAt), "%s: updated_at changed to %s", p.name, got[i].UpdatedAt)
		}

		var remaining int
		require.NoError(t, sqlx.Get(db, &remaining, `SELECT
			(SELECT COUNT(*) FROM app_config_json WHERE JSON_CONTAINS_PATH(json_value, 'one', '$.webhook_settings.failing_policies_webhook.policy_ids')) +
			(SELECT COUNT(*) FROM teams WHERE JSON_CONTAINS_PATH(config, 'one', '$.webhook_settings.failing_policies_webhook.policy_ids')) +
			(SELECT COUNT(*) FROM default_team_config_json WHERE JSON_CONTAINS_PATH(json_value, 'one', '$.webhook_settings.failing_policies_webhook.policy_ids'))`))
		require.Zero(t, remaining)

		// The rest of the settings stay.
		var destinationURL, teamWebhookEnabled, teamCFeatures string
		require.NoError(t, sqlx.Get(db, &destinationURL,
			`SELECT json_value->>'$.webhook_settings.failing_policies_webhook.destination_url' FROM app_config_json WHERE id = 1`))
		require.Equal(t, "https://example.com", destinationURL)
		require.NoError(t, sqlx.Get(db, &teamWebhookEnabled,
			`SELECT config->>'$.webhook_settings.failing_policies_webhook.enable_failing_policies_webhook' FROM teams WHERE id = 1`))
		require.Equal(t, "true", teamWebhookEnabled)
		require.NoError(t, sqlx.Get(db, &teamCFeatures, `SELECT CAST(config AS CHAR) FROM teams WHERE id = 3`))
		require.JSONEq(t, `{"features": {}}`, teamCFeatures)
		var teamBConfig *string
		require.NoError(t, sqlx.Get(db, &teamBConfig, `SELECT config FROM teams WHERE id = 2`))
		require.Nil(t, teamBConfig)
	}

	applyNext(t, db)
	check()

	// A retry after a partial run gives the same result.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261117042116(tx))
	require.NoError(t, tx.Commit())
	check()
}
