package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261105132801(t *testing.T) {
	runAgain := func(t *testing.T, db *sqlx.DB) {
		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261105132801(tx))
		require.NoError(t, tx.Commit())
	}

	t.Run("moves stored settings", func(t *testing.T) {
		db := applyUpToPrev(t)

		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value,
			'$.mdm.enable_recovery_lock_password', true,
			'$.mdm.macos_setup.require_all_software_macos', true,
			'$.mdm.macos_setup.require_all_software_windows', true
		) WHERE id = 1`)

		teams := map[string]string{
			"all":       `{"mdm": {"enable_recovery_lock_password": true, "macos_setup": {"lock_end_user_info": true, "require_all_software_macos": true, "require_all_software_windows": false}}}`,
			"recovery":  `{"mdm": {"enable_recovery_lock_password": false}}`,
			"bothnames": `{"mdm": {"macos_setup": {"require_all_software_macos": true, "macos_require_all_software": false}}}`,
			"empty":     `{}`,
			"nullmdm":   `{"mdm": null}`,
			"nullsetup": `{"mdm": {"macos_setup": null}}`,
		}
		for name, config := range teams {
			execNoErr(t, db, `INSERT INTO teams (name, config) VALUES (?, ?)`, name, config)
		}
		execNoErr(t, db, `INSERT INTO teams (name, config) VALUES ('nullconfig', NULL)`)

		applyNext(t, db)

		var appCfg struct {
			Old      bool `db:"old_keys"`
			Recovery bool `db:"recovery"`
			MacOS    bool `db:"macos"`
			Windows  bool `db:"windows"`
		}
		require.NoError(t, sqlx.Get(db, &appCfg, `SELECT
			JSON_CONTAINS_PATH(json_value, 'one', '$.mdm.enable_recovery_lock_password',
				'$.mdm.macos_setup.require_all_software_macos', '$.mdm.macos_setup.require_all_software_windows') AS old_keys,
			json_value->'$.mdm.macos_recovery_lock_password_enabled' = true AS recovery,
			json_value->'$.mdm.macos_setup.macos_require_all_software' = true AS macos,
			json_value->'$.mdm.macos_setup.windows_require_all_software' = true AS windows
			FROM app_config_json WHERE id = 1`))
		require.False(t, appCfg.Old)
		require.True(t, appCfg.Recovery)
		require.True(t, appCfg.MacOS)
		require.True(t, appCfg.Windows)

		want := map[string]string{
			"all":       `{"mdm": {"macos_recovery_lock_password_enabled": true, "macos_setup": {"lock_end_user_info": true, "macos_require_all_software": true, "windows_require_all_software": false}}}`,
			"recovery":  `{"mdm": {"macos_recovery_lock_password_enabled": false}}`,
			"bothnames": `{"mdm": {"macos_setup": {"macos_require_all_software": true}}}`,
			"empty":     `{}`,
			"nullmdm":   `{"mdm": null}`,
			"nullsetup": `{"mdm": {"macos_setup": null}}`,
		}
		checkTeams := func() {
			for name, config := range want {
				var got string
				require.NoError(t, sqlx.Get(db, &got, `SELECT config FROM teams WHERE name = ?`, name))
				require.JSONEq(t, config, got, name)
			}
			var nullConfig *string
			require.NoError(t, sqlx.Get(db, &nullConfig, `SELECT config FROM teams WHERE name = 'nullconfig'`))
			require.Nil(t, nullConfig)
		}
		checkTeams()

		// a second run finds nothing to move
		runAgain(t, db)
		checkTeams()
	})

	t.Run("app config values", func(t *testing.T) {
		db := applyUpToPrev(t)
		applyNext(t, db)

		for _, tc := range []struct{ name, mdm, want string }{
			{
				"false",
				`{"enable_recovery_lock_password": false, "macos_setup": {"require_all_software_macos": false, "require_all_software_windows": false}}`,
				`{"macos_recovery_lock_password_enabled": false, "macos_setup": {"macos_require_all_software": false, "windows_require_all_software": false}}`,
			},
			{"missing", `{"macos_setup": {"lock_end_user_info": true}}`, `{"macos_setup": {"lock_end_user_info": true}}`},
			{
				// the old name holds the setting from before the upgrade
				"both names",
				`{"enable_recovery_lock_password": true, "macos_recovery_lock_password_enabled": false, "macos_setup": {"require_all_software_windows": true, "windows_require_all_software": false}}`,
				`{"macos_recovery_lock_password_enabled": true, "macos_setup": {"windows_require_all_software": true}}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm', CAST(? AS JSON)) WHERE id = 1`, tc.mdm)
				got := func() string {
					var mdm string
					require.NoError(t, sqlx.Get(db, &mdm, `SELECT json_value->'$.mdm' FROM app_config_json WHERE id = 1`))
					return mdm
				}
				runAgain(t, db)
				require.JSONEq(t, tc.want, got())

				// a second run finds nothing to move
				runAgain(t, db)
				require.JSONEq(t, tc.want, got())
			})
		}
	})
}
