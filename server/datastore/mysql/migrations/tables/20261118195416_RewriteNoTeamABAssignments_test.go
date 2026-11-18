package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261118195416(t *testing.T) {
	db := applyUpToPrev(t)
	// "No team" in any letter case is rewritten, under both the new and the old key names, and nothing else is.
	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.apple_business', CAST(? AS JSON),
		'$.mdm.apple_business_manager', CAST(? AS JSON)) WHERE id = 1`,
		`[{"organization_name": "a", "macos_fleet": "No team", "ios_fleet": "Workstations", "ipados_fleet": "", "byod_fleet": "No team"},
		{"organization_name": "b", "macos_fleet": "no team", "ios_fleet": "Unassigned", "ipados_fleet": "No team", "byod_fleet": ""}]`,
		`[{"organization_name": "c", "macos_team": "No team", "ios_team": "Workstations", "ipados_team": "", "byod_team": ""}]`)

	applyNext(t, db)

	var raw []byte
	require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value->'$.mdm' FROM app_config_json WHERE id = 1`))
	var mdm map[string]any
	require.NoError(t, json.Unmarshal(raw, &mdm))
	require.Equal(t, []any{
		map[string]any{"organization_name": "a", "macos_fleet": "", "ios_fleet": "Workstations", "ipados_fleet": "", "byod_fleet": ""},
		map[string]any{"organization_name": "b", "macos_fleet": "", "ios_fleet": "Unassigned", "ipados_fleet": "", "byod_fleet": ""},
	}, mdm["apple_business"])
	require.Equal(t, []any{
		map[string]any{"organization_name": "c", "macos_team": "", "ios_team": "Workstations", "ipados_team": "", "byod_team": ""},
	}, mdm["apple_business_manager"])
}
