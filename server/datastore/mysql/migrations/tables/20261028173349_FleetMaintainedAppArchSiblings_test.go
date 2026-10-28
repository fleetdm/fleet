package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261028173349(t *testing.T) {
	db := applyUpToPrev(t)

	const installerInsert = `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, filename, extension, version, platform,
			 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query,
			 fleet_maintained_app_id, is_active)
		VALUES (?, ?, ?, ?, 'msi', ?, 'windows', ?, ?, ?, '', '', ?, ?)`
	script := execNoErrLastID(t, db, `INSERT INTO script_contents (contents, md5_checksum) VALUES ('echo', UNHEX(MD5('echo')))`)
	insertInstaller := func(teamID, titleID int64, version, storage string, fmaID *int64, active int) error {
		_, err := db.Exec(installerInsert, teamID, teamID, titleID, storage+".msi", version, script, script, storage, fmaID, active)
		return err
	}

	team := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('Team 1')`)
	chromeX64 := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES ('Google Chrome', 'google-chrome/windows', 'windows', 'Google Chrome')`)
	chromeARM := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES ('Google Chrome (ARM64)', 'google-chrome-arm64/windows', 'windows', 'Google Chrome')`)
	slack := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES ('Slack', 'slack/windows', 'windows', 'Slack')`)

	chromeTitle := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('Google Chrome', 'programs')`)
	slackTitle := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('Slack', 'programs')`)
	orphanTitle := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('Removed', 'programs')`)

	// Chrome x64 with two cached versions; the second one is active and pinned.
	require.NoError(t, insertInstaller(team, chromeTitle, "1.0", "chrome-1", &chromeX64, 0))
	require.NoError(t, insertInstaller(team, chromeTitle, "2.0", "chrome-2", &chromeX64, 1))
	require.NoError(t, insertInstaller(team, slackTitle, "4.0", "slack-4", &slack, 1))

	// Before the migration, the ARM64 build can't be cached at a version the x64 build already has.
	require.Error(t, insertInstaller(team, chromeTitle, "2.0", "chrome-arm-2", &chromeARM, 0))

	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (?, ?, '2.0')`, team, chromeTitle)
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (?, ?, '^4')`, team, slackTitle)
	// A pin whose installers are gone stays unattached.
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (?, ?, '1.0')`, team, orphanTitle)

	applyNext(t, db)

	pinApp := func(titleID int64) int64 {
		var fmaID int64
		require.NoError(t, db.QueryRow(`SELECT fleet_maintained_app_id FROM software_title_team_pins WHERE team_id = ? AND title_id = ?`, team, titleID).Scan(&fmaID))
		return fmaID
	}
	require.Equal(t, chromeX64, pinApp(chromeTitle))
	require.Equal(t, slack, pinApp(slackTitle))
	require.Zero(t, pinApp(orphanTitle))

	// Both builds can now be cached at the same version on one title.
	require.NoError(t, insertInstaller(team, chromeTitle, "2.0", "chrome-arm-2", &chromeARM, 1))
	// The same build still can't be cached twice at one version.
	require.Error(t, insertInstaller(team, chromeTitle, "2.0", "chrome-arm-2b", &chromeARM, 0))
	// Custom packages still dedupe by content hash.
	require.NoError(t, insertInstaller(team, slackTitle, "5.0", "custom-hash", nil, 1))
	require.Error(t, insertInstaller(team, slackTitle, "6.0", "custom-hash", nil, 1))

	// Each build gets its own pin.
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (?, ?, ?, '^2')`, team, chromeTitle, chromeARM)
	var pins int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins WHERE team_id = ? AND title_id = ?`, team, chromeTitle).Scan(&pins))
	require.Equal(t, 2, pins)
	_, err := db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (?, ?, ?, '1.0')`, team, chromeTitle, chromeARM)
	require.Error(t, err)

	// A retry after a partial failure is a no-op.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261028173349(tx))
	require.NoError(t, tx.Commit())
	require.Equal(t, chromeX64, pinApp(chromeTitle))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins WHERE team_id = ? AND title_id = ?`, team, chromeTitle).Scan(&pins))
	require.Equal(t, 2, pins)
}
