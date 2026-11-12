package tables

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261112153643(t *testing.T) {
	// Each case sets up the dark and light pairs differently, so one migration run covers two branches.
	for _, tc := range []struct {
		name, orgInfo       string
		wantDark, wantLight sql.NullString
	}{
		{
			name: "old value moves to missing or empty",
			orgInfo: `{"org_name": "Acme", "org_logo_url": "https://example.com/dark.png",
				"org_logo_url_light_background": "https://example.com/light.png", "org_logo_url_light_mode": ""}`,
			wantDark:  sql.NullString{String: "https://example.com/dark.png", Valid: true},
			wantLight: sql.NullString{String: "https://example.com/light.png", Valid: true},
		},
		{
			name: "new key wins and absent pair untouched",
			orgInfo: `{"org_name": "Acme", "org_logo_url": "https://example.com/old-dark.png",
				"org_logo_url_dark_mode": "https://example.com/dark.png"}`,
			wantDark: sql.NullString{String: "https://example.com/dark.png", Valid: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := applyUpToPrev(t)
			execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.org_info', CAST(? AS JSON)) WHERE id = 1`,
				tc.orgInfo)

			applyNext(t, db)

			var got struct {
				OldKeys   bool           `db:"old_keys"`
				OrgName   string         `db:"org_name"`
				DarkMode  sql.NullString `db:"dark_mode"`
				LightMode sql.NullString `db:"light_mode"`
			}
			require.NoError(t, sqlx.Get(db, &got, `SELECT
				JSON_CONTAINS_PATH(json_value, 'one', '$.org_info.org_logo_url', '$.org_info.org_logo_url_light_background') AS old_keys,
				json_value->>'$.org_info.org_name' AS org_name,
				json_value->>'$.org_info.org_logo_url_dark_mode' AS dark_mode,
				json_value->>'$.org_info.org_logo_url_light_mode' AS light_mode
				FROM app_config_json WHERE id = 1`))
			require.False(t, got.OldKeys)
			require.Equal(t, "Acme", got.OrgName)
			require.Equal(t, tc.wantDark, got.DarkMode)
			require.Equal(t, tc.wantLight, got.LightMode)
		})
	}
}
