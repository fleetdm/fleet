package tables

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261112153643(t *testing.T) {
	type orgLogos struct {
		OldKeys   bool           `db:"old_keys"`
		DarkMode  sql.NullString `db:"dark_mode"`
		LightMode sql.NullString `db:"light_mode"`
	}
	read := func(t *testing.T, db *sqlx.DB) orgLogos {
		t.Helper()
		var got orgLogos
		require.NoError(t, sqlx.Get(db, &got, `SELECT
			JSON_CONTAINS_PATH(json_value, 'one', '$.org_info.org_logo_url', '$.org_info.org_logo_url_light_background') AS old_keys,
			json_value->>'$.org_info.org_logo_url_dark_mode' AS dark_mode,
			json_value->>'$.org_info.org_logo_url_light_mode' AS light_mode
			FROM app_config_json WHERE id = 1`))
		return got
	}
	// setOrgInfo replaces the stored org_info with the given JSON object.
	setOrgInfo := func(t *testing.T, db *sqlx.DB, orgInfo string) {
		t.Helper()
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.org_info', CAST(? AS JSON)) WHERE id = 1`, orgInfo)
	}

	for _, tc := range []struct {
		name          string
		orgInfo       string
		wantDarkMode  sql.NullString
		wantLightMode sql.NullString
	}{
		{
			name: "old and new keys mirrored",
			orgInfo: `{"org_name": "Acme", "org_logo_url": "https://example.com/dark.png", "org_logo_url_dark_mode": "https://example.com/dark.png",
				"org_logo_url_light_background": "https://example.com/light.png", "org_logo_url_light_mode": "https://example.com/light.png"}`,
			wantDarkMode:  sql.NullString{String: "https://example.com/dark.png", Valid: true},
			wantLightMode: sql.NullString{String: "https://example.com/light.png", Valid: true},
		},
		{
			name:          "only old keys",
			orgInfo:       `{"org_name": "Acme", "org_logo_url": "https://example.com/dark.png", "org_logo_url_light_background": "https://example.com/light.png"}`,
			wantDarkMode:  sql.NullString{String: "https://example.com/dark.png", Valid: true},
			wantLightMode: sql.NullString{String: "https://example.com/light.png", Valid: true},
		},
		{
			name: "new keys empty",
			orgInfo: `{"org_name": "Acme", "org_logo_url": "https://example.com/dark.png", "org_logo_url_dark_mode": "",
				"org_logo_url_light_background": "https://example.com/light.png", "org_logo_url_light_mode": ""}`,
			wantDarkMode:  sql.NullString{String: "https://example.com/dark.png", Valid: true},
			wantLightMode: sql.NullString{String: "https://example.com/light.png", Valid: true},
		},
		{
			name: "new keys win over different old values",
			orgInfo: `{"org_name": "Acme", "org_logo_url": "https://example.com/old-dark.png", "org_logo_url_dark_mode": "https://example.com/dark.png",
				"org_logo_url_light_background": "https://example.com/old-light.png", "org_logo_url_light_mode": "https://example.com/light.png"}`,
			wantDarkMode:  sql.NullString{String: "https://example.com/dark.png", Valid: true},
			wantLightMode: sql.NullString{String: "https://example.com/light.png", Valid: true},
		},
		{
			name:          "all empty",
			orgInfo:       `{"org_name": "Acme", "org_logo_url": "", "org_logo_url_dark_mode": "", "org_logo_url_light_background": "", "org_logo_url_light_mode": ""}`,
			wantDarkMode:  sql.NullString{String: "", Valid: true},
			wantLightMode: sql.NullString{String: "", Valid: true},
		},
		{
			name:          "only new keys",
			orgInfo:       `{"org_name": "Acme", "org_logo_url_dark_mode": "https://example.com/dark.png"}`,
			wantDarkMode:  sql.NullString{String: "https://example.com/dark.png", Valid: true},
			wantLightMode: sql.NullString{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := applyUpToPrev(t)
			setOrgInfo(t, db, tc.orgInfo)

			applyNext(t, db)

			got := read(t, db)
			require.False(t, got.OldKeys)
			require.Equal(t, tc.wantDarkMode, got.DarkMode)
			require.Equal(t, tc.wantLightMode, got.LightMode)

			var orgName string
			require.NoError(t, sqlx.Get(db, &orgName, `SELECT json_value->>'$.org_info.org_name' FROM app_config_json WHERE id = 1`))
			require.Equal(t, "Acme", orgName)
		})
	}
}
