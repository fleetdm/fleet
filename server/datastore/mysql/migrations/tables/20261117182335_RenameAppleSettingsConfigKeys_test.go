package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261117182335(t *testing.T) {
	runAgain := func(t *testing.T, db *sqlx.DB) {
		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261117182335(tx))
		require.NoError(t, tx.Commit())
	}

	const oldMDM = `{
		"enable_disk_encryption": true,
		"macos_settings": {"custom_settings": [{"path": "a.mobileconfig", "labels": ["A"]}], "enable_disk_encryption": true},
		"windows_settings": {"custom_settings": [{"path": "b.xml"}], "enable_disk_encryption": false},
		"android_settings": {"custom_settings": null, "certificates": null}
	}`
	const newMDM = `{
		"enable_disk_encryption": true,
		"apple_settings": {"configuration_profiles": [{"path": "a.mobileconfig", "labels": ["A"]}], "enable_disk_encryption": true},
		"windows_settings": {"configuration_profiles": [{"path": "b.xml"}], "enable_disk_encryption": false},
		"android_settings": {"configuration_profiles": null, "certificates": null}
	}`

	t.Run("moves stored settings", func(t *testing.T) {
		db := applyUpToPrev(t)

		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm', CAST(? AS JSON)) WHERE id = 1`, oldMDM)
		execNoErr(t, db, `UPDATE default_team_config_json SET json_value = JSON_SET(json_value, '$.mdm', CAST(? AS JSON)) WHERE id = 1`, oldMDM)

		teams := map[string]string{
			"all":          `{"mdm": ` + oldMDM + `, "features": {"enable_host_users": true}}`,
			"nullvalues":   `{"mdm": {"macos_settings": {"custom_settings": null}}}`,
			"bothparents":  `{"mdm": {"macos_settings": {"custom_settings": [{"path": "old"}]}, "apple_settings": {"configuration_profiles": [{"path": "new"}]}}}`,
			"bothchildren": `{"mdm": {"windows_settings": {"custom_settings": [{"path": "old"}], "configuration_profiles": []}}}`,
			"renamed":      `{"mdm": ` + newMDM + `}`,
			"empty":        `{}`,
			"nullmdm":      `{"mdm": null}`,
			"nullobjects":  `{"mdm": {"macos_settings": null, "windows_settings": null}}`,
		}
		for name, config := range teams {
			execNoErr(t, db, `INSERT INTO teams (name, config) VALUES (?, ?)`, name, config)
		}
		execNoErr(t, db, `INSERT INTO teams (name, config) VALUES ('nullconfig', NULL)`)

		applyNext(t, db)

		want := map[string]string{
			"all":          `{"mdm": ` + newMDM + `, "features": {"enable_host_users": true}}`,
			"nullvalues":   `{"mdm": {"apple_settings": {"configuration_profiles": null}}}`,
			"bothparents":  `{"mdm": {"apple_settings": {"configuration_profiles": [{"path": "old"}]}}}`,
			"bothchildren": `{"mdm": {"windows_settings": {"configuration_profiles": [{"path": "old"}]}}}`,
			"renamed":      `{"mdm": ` + newMDM + `}`,
			"empty":        `{}`,
			"nullmdm":      `{"mdm": null}`,
			"nullobjects":  `{"mdm": {"apple_settings": null, "windows_settings": null}}`,
		}
		check := func() {
			for name, config := range want {
				var got string
				require.NoError(t, sqlx.Get(db, &got, `SELECT config FROM teams WHERE name = ?`, name))
				require.JSONEq(t, config, got, name)
			}
			var nullConfig *string
			require.NoError(t, sqlx.Get(db, &nullConfig, `SELECT config FROM teams WHERE name = 'nullconfig'`))
			require.Nil(t, nullConfig)

			for _, table := range []string{"app_config_json", "default_team_config_json"} {
				var got string
				require.NoError(t, sqlx.Get(db, &got, `SELECT json_value->'$.mdm' FROM `+table+` WHERE id = 1`))
				require.JSONEq(t, newMDM, got, table)
			}
		}
		check()

		// a second run finds nothing to move
		runAgain(t, db)
		check()
	})

	t.Run("fresh database defaults", func(t *testing.T) {
		db := applyUpToPrev(t)
		applyNext(t, db)

		for _, table := range []string{"app_config_json", "default_team_config_json"} {
			var oldKeys bool
			require.NoError(t, sqlx.Get(db, &oldKeys, `SELECT JSON_CONTAINS_PATH(json_value, 'one',
				'$.mdm.macos_settings', '$.mdm.apple_settings.custom_settings', '$.mdm.windows_settings.custom_settings',
				'$.mdm.android_settings.custom_settings') FROM `+table+` WHERE id = 1`))
			require.False(t, oldKeys, table)
		}
	})
}
