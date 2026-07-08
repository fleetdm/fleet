package tables

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260702164518(t *testing.T) {
	db := applyUpToPrev(t)

	insertTitle := func(name string, source string, upgradeCode string) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO software_titles (name, source, extension_for, upgrade_code) VALUES (?, ?, '', ?)`,
			name, source, upgradeCode)
	}

	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('sc')), '')`)
	addInstaller := func(titleID int64) {
		execNoErr(t, db, `
			INSERT INTO software_installers
				(title_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
			VALUES (?, 'app.msi', '1.0', 'windows', ?, ?, ?, '', '')`,
			titleID, scriptID, scriptID, fmt.Sprintf("storage-%d", titleID))
	}

	titleExists := func(id int64) bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_titles WHERE id = ?`, id).Scan(&n))
		return n == 1
	}
	upgradeCodeOf := func(id int64) string {
		var uc sql.NullString
		require.NoError(t, db.QueryRow(`SELECT upgrade_code FROM software_titles WHERE id = ?`, id).Scan(&uc))
		return uc.String
	}

	const aircallUC = "{9F7A3B21-4C5D-4E6F-8A9B-0C1D2E3F4A5B}"

	// Scenario 1 (the bug, should be fixed): a host reported "Aircall" with an upgrade code (no
	// installer), and the FMA/gitops add created a second "Aircall" with an empty upgrade code and
	// an installer. Reference rows on the drop side must follow the merge to the keep title.
	aircallDrop := insertTitle("Aircall", "programs", aircallUC)
	aircallKeep := insertTitle("Aircall", "programs", "")
	addInstaller(aircallKeep)
	execNoErr(t, db, `INSERT INTO software (name, source, checksum, title_id) VALUES ('Aircall', 'programs', UNHEX(MD5('aircall-sw')), ?)`, aircallDrop)
	execNoErr(t, db, `INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename) VALUES (0, ?, 'aircall-icon', 'i.png')`, aircallDrop)

	// Scenario 2 (not the bug): two distinct programs share a name, neither has an installer, so
	// there's nothing an admin manages here to merge into. Leave both alone.
	bonjourEmpty := insertTitle("Bonjour", "programs", "")
	bonjourCode := insertTitle("Bonjour", "programs", "{11111111-1111-1111-1111-111111111111}")

	// Scenario 3 (mirror of the bug, should be fixed): the installer is on the upgrade-code title
	// and a host-reported empty-code title of the same name has no installer. Merge the empty one
	// into the installer-backed title.
	const slackUC = "{22222222-2222-2222-2222-222222222222}"
	slackEmpty := insertTitle("Slack", "programs", "")
	slackCode := insertTitle("Slack", "programs", slackUC)
	addInstaller(slackCode)
	execNoErr(t, db, `INSERT INTO software (name, source, checksum, title_id) VALUES ('Slack', 'programs', UNHEX(MD5('slack-sw')), ?)`, slackEmpty)

	// Scenario 4 (ambiguous, should be skipped): one installer-backed empty title but two different
	// upgrade-code titles. We can't tell which code is right, so don't touch anything.
	chromeKeep := insertTitle("Chrome", "programs", "")
	addInstaller(chromeKeep)
	chromeCode1 := insertTitle("Chrome", "programs", "{33333333-3333-3333-3333-333333333333}")
	chromeCode2 := insertTitle("Chrome", "programs", "{44444444-4444-4444-4444-444444444444}")

	// Scenario 5 (not the bug): a macOS title sharing the name must be untouched by the
	// programs-only migration.
	aircallApps := insertTitle("Aircall", "apps", "")

	// Scenario 6 (not the bug): a lone installer-backed title with no host-reported partner stays
	// empty; there's no upgrade code to adopt.
	zoomSolo := insertTitle("Zoom", "programs", "")
	addInstaller(zoomSolo)

	// Scenario 7 (both installer-backed, same team, should be skipped): the same team has an installer
	// under both titles, so moving would collide. Leave both alone.
	const onePassUC = "{55555555-5555-5555-5555-555555555555}"
	onePassEmpty := insertTitle("1Password", "programs", "")
	addInstaller(onePassEmpty)
	onePassCode := insertTitle("1Password", "programs", onePassUC)
	addInstaller(onePassCode)

	// Scenario 8 (both installer-backed, different teams, should be fixed): the same program is
	// installed under two titles across different teams. No team owns both, so the empty title's
	// installer moves onto the upgrade-code title.
	const dockerUC = "{77777777-7777-7777-7777-777777777777}"
	dockerCode := insertTitle("Docker", "programs", dockerUC)
	addInstaller(dockerCode) // team 0
	dockerEmpty := insertTitle("Docker", "programs", "")
	execNoErr(t, db, `
		INSERT INTO software_installers
			(title_id, global_or_team_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
		VALUES (?, 1, 'app.msi', '1.0', 'windows', ?, ?, 'docker-team1', '', '')`,
		dockerEmpty, scriptID, scriptID)

	applyNext(t, db)

	// Scenario 1: the drop title is gone, the keep title absorbed the upgrade code, and the
	// software + icon rows now point at the keep title.
	require.False(t, titleExists(aircallDrop), "host-reported duplicate title should be deleted")
	require.True(t, titleExists(aircallKeep))
	require.Equal(t, aircallUC, upgradeCodeOf(aircallKeep), "kept title should adopt the host's upgrade code")

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_titles WHERE name = 'Aircall' AND source = 'programs'`).Scan(&count))
	require.Equal(t, 1, count, "only one Aircall programs title should remain")

	var swTitleID int64
	require.NoError(t, db.QueryRow(`SELECT title_id FROM software WHERE checksum = UNHEX(MD5('aircall-sw'))`).Scan(&swTitleID))
	require.Equal(t, aircallKeep, swTitleID)

	var iconTitleID int64
	require.NoError(t, db.QueryRow(`SELECT software_title_id FROM software_title_icons WHERE storage_id = 'aircall-icon'`).Scan(&iconTitleID))
	require.Equal(t, aircallKeep, iconTitleID)

	// Scenario 2: untouched.
	require.True(t, titleExists(bonjourEmpty))
	require.True(t, titleExists(bonjourCode))
	require.Empty(t, upgradeCodeOf(bonjourEmpty))

	// Scenario 3: the empty host title is merged into the code+installer title, which keeps its
	// upgrade code, and the software row follows.
	require.False(t, titleExists(slackEmpty), "empty host-reported title should be deleted")
	require.True(t, titleExists(slackCode))
	require.Equal(t, slackUC, upgradeCodeOf(slackCode), "installer title keeps its upgrade code")

	var slackSwTitleID int64
	require.NoError(t, db.QueryRow(`SELECT title_id FROM software WHERE checksum = UNHEX(MD5('slack-sw'))`).Scan(&slackSwTitleID))
	require.Equal(t, slackCode, slackSwTitleID)

	// Scenario 4: untouched, keep title still has no upgrade code.
	require.True(t, titleExists(chromeKeep))
	require.True(t, titleExists(chromeCode1))
	require.True(t, titleExists(chromeCode2))
	require.Empty(t, upgradeCodeOf(chromeKeep))

	// Scenario 5: the macOS title survives.
	require.True(t, titleExists(aircallApps))

	// Scenario 6: still empty, nothing merged into it.
	require.True(t, titleExists(zoomSolo))
	require.Empty(t, upgradeCodeOf(zoomSolo))

	// Scenario 7: same team under both titles, so both survive untouched.
	require.True(t, titleExists(onePassEmpty))
	require.True(t, titleExists(onePassCode))
	require.Empty(t, upgradeCodeOf(onePassEmpty))
	require.Equal(t, onePassUC, upgradeCodeOf(onePassCode))

	// Scenario 8: the empty title is merged into the upgrade-code title, and both teams' installers
	// now sit on it.
	require.False(t, titleExists(dockerEmpty), "empty title should be deleted")
	require.True(t, titleExists(dockerCode))
	require.Equal(t, dockerUC, upgradeCodeOf(dockerCode))

	var dockerInstallers int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_installers WHERE title_id = ?`, dockerCode).Scan(&dockerInstallers))
	require.Equal(t, 2, dockerInstallers, "both teams' installers should sit on the kept title")
}
