package tables

import (
	"strconv"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261105133221(t *testing.T) {
	type windowsMDM struct {
		OldKeys   bool  `db:"old_keys"`
		Enabled   *bool `db:"enabled"`
		Migration *bool `db:"migration"`
	}
	read := func(t *testing.T, db *sqlx.DB) windowsMDM {
		t.Helper()
		var got windowsMDM
		require.NoError(t, sqlx.Get(db, &got, `SELECT
			JSON_CONTAINS_PATH(json_value, 'one', '$.mdm.windows_enabled_and_configured', '$.mdm.windows_migration_enabled') AS old_keys,
			json_value->'$.mdm.windows_mdm_enabled' = true AS enabled,
			json_value->'$.mdm.windows_mdm_migration_enabled' = true AS migration
			FROM app_config_json WHERE id = 1`))
		return got
	}

	for _, tc := range []struct {
		name               string
		enabled, migration bool
	}{
		{"both on", true, true},
		{"only MDM on", true, false},
		{"both off", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := applyUpToPrev(t)
			// A value already under the new name loses to the old key.
			// CAST keeps the values JSON booleans; a bound Go bool would be stored as the number 1.
			execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value,
				'$.mdm.windows_enabled_and_configured', CAST(? AS JSON), '$.mdm.windows_migration_enabled', CAST(? AS JSON),
				'$.mdm.windows_mdm_enabled', false, '$.mdm.windows_mdm_migration_enabled', false) WHERE id = 1`,
				strconv.FormatBool(tc.enabled), strconv.FormatBool(tc.migration))

			applyNext(t, db)

			got := read(t, db)
			require.False(t, got.OldKeys)
			require.NotNil(t, got.Enabled)
			require.NotNil(t, got.Migration)
			require.Equal(t, tc.enabled, *got.Enabled)
			require.Equal(t, tc.migration, *got.Migration)
		})
	}

	t.Run("neither key", func(t *testing.T) {
		db := applyUpToPrev(t)
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_REMOVE(json_value,
			'$.mdm.windows_enabled_and_configured', '$.mdm.windows_migration_enabled',
			'$.mdm.windows_mdm_enabled', '$.mdm.windows_mdm_migration_enabled') WHERE id = 1`)

		applyNext(t, db)

		got := read(t, db)
		require.False(t, got.OldKeys)
		require.Nil(t, got.Enabled)
		require.Nil(t, got.Migration)
	})
}
