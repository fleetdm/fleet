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
			// team2 has no bootstrap package, so it can't get the setting
			execNoErr(t, db, `INSERT INTO mdm_apple_bootstrap_packages (team_id, name, sha256, token) VALUES
				(0, 'no-team.pkg', UNHEX(SHA2('a', 256)), 'token-0'),
				(1, 'team1.pkg', UNHEX(SHA2('b', 256)), 'token-1')`)

			applyNext(t, db)

			var appCfg bool
			require.NoError(t, db.QueryRow(`SELECT COALESCE(JSON_EXTRACT(json_value, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration') = CAST('true' AS JSON), false)
				FROM app_config_json WHERE id = 1`).Scan(&appCfg))
			require.Equal(t, tc.want, appCfg)

			var enabledTeamIDs []uint
			rows, err := db.Query(`SELECT id FROM teams
				WHERE JSON_EXTRACT(config, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration') = CAST('true' AS JSON)`)
			require.NoError(t, err)
			defer rows.Close()
			for rows.Next() {
				var id uint
				require.NoError(t, rows.Scan(&id))
				enabledTeamIDs = append(enabledTeamIDs, id)
			}
			require.NoError(t, rows.Err())
			if tc.want {
				require.Equal(t, []uint{1}, enabledTeamIDs)
			} else {
				require.Empty(t, enabledTeamIDs)
			}

			// other settings are left untouched
			var pkg string
			require.NoError(t, db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(config, '$.mdm.macos_setup.bootstrap_package')) FROM teams WHERE id = 1`).Scan(&pkg))
			require.Equal(t, "pkg", pkg)
		})
	}
}

func TestUp_20261029070851_NoTeamWithoutPackage(t *testing.T) {
	// t.Setenv can't be used, applyUpToPrev marks the test as parallel
	require.NoError(t, os.Setenv("FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION", "1"))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv("FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION")) })

	db := applyUpToPrev(t)
	execNoErr(t, db, `INSERT INTO teams (id, name, config) VALUES (1, 'team1', '{"mdm": {"macos_setup": {}}}')`)
	execNoErr(t, db, `INSERT INTO mdm_apple_bootstrap_packages (team_id, name, sha256, token) VALUES (1, 'team1.pkg', UNHEX(SHA2('b', 256)), 'token-1')`)

	applyNext(t, db)

	var appCfg bool
	require.NoError(t, db.QueryRow(`SELECT COALESCE(JSON_EXTRACT(json_value, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration') = CAST('true' AS JSON), false)
		FROM app_config_json WHERE id = 1`).Scan(&appCfg))
	require.False(t, appCfg)

	var teamEnabled bool
	require.NoError(t, db.QueryRow(`SELECT JSON_EXTRACT(config, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration') = CAST('true' AS JSON)
		FROM teams WHERE id = 1`).Scan(&teamEnabled))
	require.True(t, teamEnabled)
}
