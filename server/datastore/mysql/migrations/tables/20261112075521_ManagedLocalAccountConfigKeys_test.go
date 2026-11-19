package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261112075521(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value,
		'$.mdm.setup_experience.enable_managed_local_account', true,
		'$.mdm.setup_experience.end_user_local_account_type', 'standard',
		'$.mdm.windows_settings.enable_managed_local_account', true) WHERE id = 1`)

	teams := map[string]string{
		"all": `{"mdm": {"setup_experience": {"enable_managed_local_account": true, "end_user_local_account_type": "none",
			"enable_end_user_authentication": true}, "windows_settings": {"enable_managed_local_account": true, "enable_disk_encryption": false}}}`,
		"off":          `{"mdm": {"setup_experience": {"enable_managed_local_account": false, "end_user_local_account_type": "admin"}, "windows_settings": {"enable_managed_local_account": false}}}`,
		"nullvalues":   `{"mdm": {"setup_experience": {"enable_managed_local_account": null}, "windows_settings": {"enable_managed_local_account": null}}}`,
		"nosetup":      `{"mdm": {"windows_settings": {"enable_managed_local_account": true}}}`,
		"nullsetup":    `{"mdm": {"setup_experience": null, "windows_settings": {"enable_managed_local_account": true}}}`,
		"migrated":     `{"mdm": {"setup_experience": {"macos_managed_local_account_enabled": true, "windows_managed_local_account_enabled": true}, "windows_settings": {}}}`,
		"empty":        `{}`,
		"nullmdm":      `{"mdm": null}`,
		"nowinsetting": `{"mdm": {"windows_settings": {"enable_disk_encryption": true}}}`,
	}
	for name, config := range teams {
		execNoErr(t, db, `INSERT INTO teams (name, config) VALUES (?, ?)`, name, config)
	}

	applyNext(t, db)

	want := map[string]string{
		"all": `{"mdm": {"setup_experience": {"macos_managed_local_account_enabled": true, "macos_end_user_local_account_type": "none",
			"enable_end_user_authentication": true, "windows_managed_local_account_enabled": true}, "windows_settings": {"enable_disk_encryption": false}}}`,
		"off":          `{"mdm": {"setup_experience": {"macos_managed_local_account_enabled": false, "macos_end_user_local_account_type": "admin", "windows_managed_local_account_enabled": false}, "windows_settings": {}}}`,
		"nullvalues":   `{"mdm": {"setup_experience": {"macos_managed_local_account_enabled": null, "windows_managed_local_account_enabled": null}, "windows_settings": {}}}`,
		"nosetup":      `{"mdm": {"setup_experience": {"windows_managed_local_account_enabled": true}, "windows_settings": {}}}`,
		"nullsetup":    `{"mdm": {"setup_experience": {"windows_managed_local_account_enabled": true}, "windows_settings": {}}}`,
		"migrated":     teams["migrated"],
		"empty":        `{}`,
		"nullmdm":      `{"mdm": null}`,
		"nowinsetting": teams["nowinsetting"],
	}
	check := func(t *testing.T) {
		var got struct {
			Old   bool   `db:"old_key"`
			Setup string `db:"setup"`
		}
		require.NoError(t, sqlx.Get(db, &got, `SELECT JSON_CONTAINS_PATH(json_value, 'one',
			'$.mdm.setup_experience.enable_managed_local_account', '$.mdm.setup_experience.end_user_local_account_type',
			'$.mdm.windows_settings.enable_managed_local_account') AS old_key,
			JSON_OBJECT(
				'mac', json_value->'$.mdm.setup_experience.macos_managed_local_account_enabled',
				'type', json_value->'$.mdm.setup_experience.macos_end_user_local_account_type',
				'win', json_value->'$.mdm.setup_experience.windows_managed_local_account_enabled') AS setup
			FROM app_config_json WHERE id = 1`))
		require.False(t, got.Old)
		require.JSONEq(t, `{"mac": true, "type": "standard", "win": true}`, got.Setup)

		for name, config := range want {
			var got string
			require.NoError(t, sqlx.Get(db, &got, `SELECT config FROM teams WHERE name = ?`, name))
			require.JSONEq(t, config, got, name)
		}
	}
	check(t)

	t.Run("a second run changes nothing", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261112075521(tx))
		require.NoError(t, tx.Commit())
		check(t)
	})
}
