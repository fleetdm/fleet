package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261111085717(t *testing.T) {
	db := applyUpToPrev(t)

	// Mixes null, boolean, and string values to check each keeps its type. scripts sits outside mdm and must not be
	// renamed even though its nested "script" name is.
	const oldSetup = `{"bootstrap_package": "bootstrap.pkg", "macos_setup_assistant": null,
		"enable_release_device_manually": true, "manual_agent_install": false, "script": "setup.sh",
		"enable_end_user_authentication": true, "software": null}`
	const newSetup = `{"macos_bootstrap_package": "bootstrap.pkg", "apple_setup_assistant": null,
		"apple_enable_release_device_manually": true, "macos_manual_agent_install": false, "macos_script": "setup.sh",
		"enable_end_user_authentication": true, "software": null}`

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_REMOVE(JSON_SET(json_value, '$.mdm.macos_setup', CAST(? AS JSON)),
		'$.mdm.setup_experience') WHERE id = 1`, oldSetup)
	execNoErr(t, db, `UPDATE default_team_config_json SET json_value = JSON_SET(json_value, '$.mdm.macos_setup', CAST(? AS JSON))
		WHERE id = 1`, oldSetup)

	teams := map[string]string{
		"all":        `{"mdm": {"macos_setup": ` + oldSetup + `}, "scripts": ["a.sh"]}`,
		"bothnames":  `{"mdm": {"macos_setup": {"bootstrap_package": "old.pkg"}, "setup_experience": {"macos_bootstrap_package": "new.pkg"}}}`,
		"outoforder": `{"mdm": {"macos_setup": {"require_all_software_macos": true, "require_all_software_windows": false}}}`,
		"migrated":   `{"mdm": {"setup_experience": {"macos_bootstrap_package": "new.pkg"}}}`,
		"empty":      `{}`,
		"nullmdm":    `{"mdm": null}`,
		"nullsetup":  `{"mdm": {"macos_setup": null}}`,
	}
	for name, config := range teams {
		execNoErr(t, db, `INSERT INTO teams (name, config) VALUES (?, ?)`, name, config)
	}
	execNoErr(t, db, `INSERT INTO teams (name, config) VALUES ('nullconfig', NULL)`)

	applyNext(t, db)

	want := map[string]string{
		"all": `{"mdm": {"setup_experience": ` + newSetup + `}, "scripts": ["a.sh"]}`,
		// The old key wins, the same as the other Fleet 5 renames.
		"bothnames": `{"mdm": {"setup_experience": {"macos_bootstrap_package": "old.pkg"}}}`,
		"migrated":  `{"mdm": {"setup_experience": {"macos_bootstrap_package": "new.pkg"}}}`,
		"empty":     `{}`,
		"nullmdm":   `{"mdm": null}`,
		"nullsetup": `{"mdm": {"setup_experience": null}}`,
	}
	check := func(t *testing.T) {
		for _, table := range []string{"app_config_json", "default_team_config_json"} {
			var got struct {
				Old   bool   `db:"old_key"`
				Setup string `db:"setup"`
			}
			require.NoError(t, sqlx.Get(db, &got, `SELECT JSON_CONTAINS_PATH(json_value, 'one', '$.mdm.macos_setup') AS old_key,
				json_value->'$.mdm.setup_experience' AS setup FROM `+table+` WHERE id = 1`))
			require.False(t, got.Old, table)
			require.JSONEq(t, newSetup, got.Setup, table)
		}

		for name, config := range want {
			var got string
			require.NoError(t, sqlx.Get(db, &got, `SELECT config FROM teams WHERE name = ?`, name))
			require.JSONEq(t, config, got, name)
		}
		var nullConfig *string
		require.NoError(t, sqlx.Get(db, &nullConfig, `SELECT config FROM teams WHERE name = 'nullconfig'`))
		require.Nil(t, nullConfig)
	}
	check(t)

	t.Run("a second run changes nothing", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261111085717(tx))
		require.NoError(t, tx.Commit())
		check(t)
	})
}
