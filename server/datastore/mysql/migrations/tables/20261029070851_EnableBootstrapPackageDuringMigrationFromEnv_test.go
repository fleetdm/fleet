package tables

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261029070851(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
		want bool
	}{
		{"unset", "", false},
		{"zero", "0", false},
		{"one", "1", true},
		{"true", "TRUE", true},
	} {
		// the subtest name is used as the database name
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv can't be used, applyUpToPrev marks the test as parallel
			require.NoError(t, os.Setenv("FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION", tc.env))
			t.Cleanup(func() { require.NoError(t, os.Unsetenv("FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION")) })

			db := applyUpToPrev(t)
			execNoErr(t, db, `INSERT INTO teams (id, name, config) VALUES
				(1, 'team1', '{"mdm": {"macos_setup": {"bootstrap_package": "pkg"}}}'),
				(2, 'team2', '{"mdm": {"macos_setup": {"macos_enable_bootstrap_package_during_migration": false}}}')`)

			applyNext(t, db)

			var appCfg bool
			require.NoError(t, db.QueryRow(`SELECT COALESCE(JSON_EXTRACT(json_value, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration') = CAST('true' AS JSON), false)
				FROM app_config_json WHERE id = 1`).Scan(&appCfg))
			require.Equal(t, tc.want, appCfg)

			var teamsEnabled int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM teams
				WHERE JSON_EXTRACT(config, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration') = CAST('true' AS JSON)`).Scan(&teamsEnabled))
			if tc.want {
				require.Equal(t, 2, teamsEnabled)
			} else {
				require.Zero(t, teamsEnabled)
			}

			// other settings are left untouched
			var pkg string
			require.NoError(t, db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(config, '$.mdm.macos_setup.bootstrap_package')) FROM teams WHERE id = 1`).Scan(&pkg))
			require.Equal(t, "pkg", pkg)
		})
	}
}
