package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	dockerDesktopStaleBundleID = "com.electron.dockerdesktop"
	dockerDesktopBundleID      = "com.docker.docker"
)

func insertDockerDesktopInstaller(t *testing.T, db *sqlx.DB, titleID, teamID int64, storageID string) int64 {
	scriptID := execNoErrLastID(t, db,
		`INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5(?)), '')`, storageID)
	return execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(title_id, global_or_team_id, filename, version, platform,
			 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
		VALUES (?, ?, 'Docker.dmg', '4.86.0', 'darwin', ?, ?, ?, '', '')`,
		titleID, teamID, scriptID, scriptID, storageID)
}

func dockerDesktopTitleBundleID(t *testing.T, db *sqlx.DB, titleID int64) string {
	var bundleID string
	require.NoError(t, db.Get(&bundleID,
		`SELECT bundle_identifier FROM software_titles WHERE id = ?`, titleID))
	return bundleID
}

func dockerDesktopInstallerTitleID(t *testing.T, db *sqlx.DB, installerID int64) int64 {
	var titleID int64
	require.NoError(t, db.Get(&titleID,
		`SELECT title_id FROM software_installers WHERE id = ?`, installerID))
	return titleID
}

func dockerDesktopTitleExists(t *testing.T, db *sqlx.DB, titleID int64) bool {
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM software_titles WHERE id = ?`, titleID))
	return count > 0
}

// No inventory ever reported Docker Desktop, so the stale title is relabeled in place and
// the installer stays attached to it.
func TestUp_20260810152924_RelabelsInPlace(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker Desktop', 'apps', ?)`,
		dockerDesktopStaleBundleID)
	installerID := insertDockerDesktopInstaller(t, db, staleTitleID, 0, "storage-stale")

	applyNext(t, db)

	require.True(t, dockerDesktopTitleExists(t, db, staleTitleID))
	require.Equal(t, dockerDesktopBundleID, dockerDesktopTitleBundleID(t, db, staleTitleID))
	require.Equal(t, staleTitleID, dockerDesktopInstallerTitleID(t, db, installerID))
}

// Inventory already created the com.docker.docker title, so the stale title is merged into
// it: the installer and its install history move over and the stale title is dropped.
func TestUp_20260810152924_MergesIntoInventoryTitle(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker Desktop', 'apps', ?)`,
		dockerDesktopStaleBundleID)
	targetTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker', 'apps', ?)`,
		dockerDesktopBundleID)

	installerID := insertDockerDesktopInstaller(t, db, staleTitleID, 0, "storage-stale")

	// Install history recorded against the stale title.
	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (hostname, osquery_host_id, node_key) VALUES ('h1', 'oh1', 'nk1')`)
	execNoErr(t, db, `
		INSERT INTO host_software_installs
			(host_id, execution_id, software_installer_id, software_title_id, install_script_exit_code)
		VALUES (?, 'exec-1', ?, ?, 0)`,
		hostID, installerID, staleTitleID)

	// A per-team setting that the target title does not have yet, so it moves.
	execNoErr(t, db,
		`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '4.86.0')`,
		staleTitleID)

	// Re-pointing a foreign key must not restamp updated_at on records that only moved.
	execNoErr(t, db,
		`UPDATE software_installers SET updated_at = '2026-01-15 12:00:00' WHERE id = ?`, installerID)
	execNoErr(t, db,
		`UPDATE host_software_installs SET updated_at = '2026-01-15 12:00:00' WHERE execution_id = 'exec-1'`)

	applyNext(t, db)

	require.False(t, dockerDesktopTitleExists(t, db, staleTitleID))
	require.Equal(t, targetTitleID, dockerDesktopInstallerTitleID(t, db, installerID))

	var historyTitleID int64
	require.NoError(t, db.Get(&historyTitleID,
		`SELECT software_title_id FROM host_software_installs WHERE execution_id = 'exec-1'`))
	require.Equal(t, targetTitleID, historyTitleID)

	var pinnedTitleID int64
	require.NoError(t, db.Get(&pinnedTitleID,
		`SELECT title_id FROM software_title_team_pins WHERE team_id = 0`))
	require.Equal(t, targetTitleID, pinnedTitleID)

	want := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	for _, q := range []string{
		`SELECT updated_at FROM software_installers WHERE id = ?`,
		`SELECT updated_at FROM host_software_installs WHERE software_installer_id = ?`,
	} {
		var updatedAt time.Time
		require.NoError(t, db.Get(&updatedAt, q, installerID))
		require.Equal(t, want, updatedAt.UTC(), q)
	}
}

// A per-team setting that already exists on the target title cannot move; the duplicate is
// dropped rather than blocking the merge.
func TestUp_20260810152924_DropsDuplicateTeamSettings(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker Desktop', 'apps', ?)`,
		dockerDesktopStaleBundleID)
	targetTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker', 'apps', ?)`,
		dockerDesktopBundleID)

	insertDockerDesktopInstaller(t, db, staleTitleID, 0, "storage-stale")

	// Team 0 has a pin on both titles, so the stale one is a duplicate.
	execNoErr(t, db,
		`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '4.86.0'), (0, ?, '4.86.0')`,
		staleTitleID, targetTitleID)

	applyNext(t, db)

	require.False(t, dockerDesktopTitleExists(t, db, staleTitleID))

	var pins []int64
	require.NoError(t, db.Select(&pins,
		`SELECT title_id FROM software_title_team_pins WHERE team_id = 0`))
	require.Equal(t, []int64{targetTitleID}, pins)
}

// A team that already has an installer on the target title is left alone: moving the FMA
// installer there would put two installers on one title for one team.
func TestUp_20260810152924_SkipsTeamsWithExistingInstaller(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker Desktop', 'apps', ?)`,
		dockerDesktopStaleBundleID)
	targetTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker', 'apps', ?)`,
		dockerDesktopBundleID)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1')`)

	// Team 1 uploaded its own Docker package and also added the FMA.
	insertDockerDesktopInstaller(t, db, targetTitleID, teamID, "storage-custom")
	blockedInstallerID := insertDockerDesktopInstaller(t, db, staleTitleID, teamID, "storage-fma-team")

	// The global installer has no conflict, so it still moves.
	movableInstallerID := insertDockerDesktopInstaller(t, db, staleTitleID, 0, "storage-fma-global")

	applyNext(t, db)

	require.Equal(t, staleTitleID, dockerDesktopInstallerTitleID(t, db, blockedInstallerID))
	require.Equal(t, targetTitleID, dockerDesktopInstallerTitleID(t, db, movableInstallerID))

	// The stale title survives because an installer still depends on it; deleting it would
	// null out that installer's title_id.
	require.True(t, dockerDesktopTitleExists(t, db, staleTitleID))
}

// Nothing to do when the FMA was never added.
func TestUp_20260810152924_NoOpWithoutStaleTitle(t *testing.T) {
	db := applyUpToPrev(t)

	targetTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Docker', 'apps', ?)`,
		dockerDesktopBundleID)

	applyNext(t, db)

	require.True(t, dockerDesktopTitleExists(t, db, targetTitleID))
	require.Equal(t, dockerDesktopBundleID, dockerDesktopTitleBundleID(t, db, targetTitleID))
}
