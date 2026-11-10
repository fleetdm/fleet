package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261110062427(t *testing.T) {
	readMDM := func(t *testing.T, db *sqlx.DB) map[string]any {
		t.Helper()
		var raw []byte
		require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value->'$.mdm' FROM app_config_json WHERE id = 1`))
		var mdm map[string]any
		require.NoError(t, json.Unmarshal(raw, &mdm))
		return mdm
	}
	oldKeys := []string{"apple_business_manager", "apple_bm_enabled_and_configured", "apple_bm_terms_expired", "apple_bm_default_team"}

	t.Run("old keys", func(t *testing.T) {
		db := applyUpToPrev(t)
		// A value already under a new name loses to the old key. The org name holds a would-be key to show the text
		// replace only touches keys.
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value,
			'$.mdm.apple_business_manager', CAST(? AS JSON),
			'$.mdm.apple_bm_enabled_and_configured', true,
			'$.mdm.apple_bm_terms_expired', true,
			'$.mdm.apple_bm_default_team', 'Workstations',
			'$.mdm.apple_business', JSON_ARRAY(),
			'$.mdm.apple_business_enabled_and_configured', false) WHERE id = 1`,
			`[{"organization_name": "org \"macos_team\": x", "default": true, "macos_team": "a", "ios_team": "b", "ipados_team": "c", "byod_team": ""}]`)

		applyNext(t, db)

		mdm := readMDM(t, db)
		for _, k := range oldKeys {
			require.NotContains(t, mdm, k)
		}
		require.Equal(t, true, mdm["apple_business_enabled_and_configured"])
		require.Equal(t, true, mdm["apple_business_terms_expired"])
		require.Equal(t, []any{map[string]any{
			"organization_name": `org "macos_team": x`,
			"default":           true,
			"macos_fleet":       "a",
			"ios_fleet":         "b",
			"ipados_fleet":      "c",
			"byod_fleet":        "",
		}}, mdm["apple_business"])
	})

	t.Run("no old keys", func(t *testing.T) {
		db := applyUpToPrev(t)
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_REMOVE(json_value,
			'$.mdm.apple_business_manager', '$.mdm.apple_bm_enabled_and_configured', '$.mdm.apple_bm_terms_expired',
			'$.mdm.apple_bm_default_team', '$.mdm.apple_business', '$.mdm.apple_business_enabled_and_configured',
			'$.mdm.apple_business_terms_expired') WHERE id = 1`)

		applyNext(t, db)

		mdm := readMDM(t, db)
		for _, k := range append(oldKeys, "apple_business", "apple_business_enabled_and_configured", "apple_business_terms_expired") {
			require.NotContains(t, mdm, k)
		}
	})

	t.Run("null apple_business_manager", func(t *testing.T) {
		db := applyUpToPrev(t)
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.apple_business_manager', CAST('null' AS JSON)) WHERE id = 1`)

		applyNext(t, db)

		mdm := readMDM(t, db)
		require.NotContains(t, mdm, "apple_business_manager")
		require.Contains(t, mdm, "apple_business")
		require.Nil(t, mdm["apple_business"])
	})
}
