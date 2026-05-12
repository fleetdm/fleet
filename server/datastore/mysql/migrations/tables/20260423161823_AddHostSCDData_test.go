package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260423161823(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed an existing team config that lacks historical_data — simulates a
	// pre-change deployment so the backfill's JSON_MERGE_PATCH path is exercised.
	execNoErr(t, db, `
		INSERT INTO teams (name, description, config)
		VALUES (?, ?, ?)
	`, "team1", "test", `{"features":{"enable_host_users":true}}`)

	applyNext(t, db)

	// AppConfig backfill: features.historical_data.{uptime,vulnerabilities} = true.
	var raw json.RawMessage
	require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value FROM app_config_json LIMIT 1;`))

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(raw, &cfg))
	features, ok := cfg["features"].(map[string]any)
	require.True(t, ok, "AppConfig features object present")
	hd, ok := features["historical_data"].(map[string]any)
	require.True(t, ok, "AppConfig features.historical_data present")
	require.Equal(t, true, hd["uptime"], "AppConfig uptime defaulted true")
	require.Equal(t, true, hd["vulnerabilities"], "AppConfig vulnerabilities defaulted true")

	// Team config backfill: same path under teams.config.
	var teamRaw json.RawMessage
	require.NoError(t, sqlx.Get(db, &teamRaw, `SELECT config FROM teams WHERE name = 'team1' LIMIT 1;`))

	var teamCfg map[string]any
	require.NoError(t, json.Unmarshal(teamRaw, &teamCfg))
	teamFeatures, ok := teamCfg["features"].(map[string]any)
	require.True(t, ok, "team features object present")
	teamHD, ok := teamFeatures["historical_data"].(map[string]any)
	require.True(t, ok, "team features.historical_data present after backfill")
	require.Equal(t, true, teamHD["uptime"], "team uptime defaulted true")
	require.Equal(t, true, teamHD["vulnerabilities"], "team vulnerabilities defaulted true")

	// Pre-existing fields under features.* SHALL be preserved by JSON_MERGE_PATCH.
	require.Equal(t, true, teamFeatures["enable_host_users"], "JSON_MERGE_PATCH preserved enable_host_users")
}
