package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	fourKVideoDownloaderStaleBundleID = "com.4kdownload.ApplicationDirectories"
	fourKVideoDownloaderBundleID      = "com.openmedia.4kvideodownloader"
)

func insertFourKVideoDownloaderTitle(t *testing.T, db *sqlx.DB, bundleID string) int64 {
	return execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('4K Video Downloader', 'apps', ?)`,
		bundleID)
}

func insertFourKVideoDownloaderInstaller(t *testing.T, db *sqlx.DB, titleID, teamID int64, storageID string) int64 {
	scriptID := execNoErrLastID(t, db,
		`INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5(?)), '')`, storageID)
	return execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(title_id, global_or_team_id, filename, version, platform,
			 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
		VALUES (?, ?, '4kvideodownloader_4.33.5_x64.dmg', '4.33.5', 'darwin', ?, ?, ?, '', '')`,
		titleID, teamID, scriptID, scriptID, storageID)
}

func insertFourKVideoDownloaderPolicy(t *testing.T, db *sqlx.DB, name, query string, installerID int64) int64 {
	return execNoErrLastID(t, db,
		`INSERT INTO policies (name, query, description, checksum, team_id, software_installer_id) VALUES (?, ?, '', UNHEX(MD5(?)), 0, ?)`,
		name, query, name, installerID)
}

func fourKVideoDownloaderTitleBundleID(t *testing.T, db *sqlx.DB, titleID int64) string {
	var bundleID string
	require.NoError(t, db.Get(&bundleID, `SELECT bundle_identifier FROM software_titles WHERE id = ?`, titleID))
	return bundleID
}

func fourKVideoDownloaderInstallerTitleID(t *testing.T, db *sqlx.DB, installerID int64) int64 {
	var titleID int64
	require.NoError(t, db.Get(&titleID, `SELECT title_id FROM software_installers WHERE id = ?`, installerID))
	return titleID
}

func fourKVideoDownloaderTitleExists(t *testing.T, db *sqlx.DB, titleID int64) bool {
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM software_titles WHERE id = ?`, titleID))
	return count > 0
}

func fourKVideoDownloaderPolicyQuery(t *testing.T, db *sqlx.DB, policyID int64) string {
	var query string
	require.NoError(t, db.Get(&query, `SELECT query FROM policies WHERE id = ?`, policyID))
	return query
}

// No host reported the app, so the stale title is relabeled in place, and the generated
// automatic-install policy query is rewritten while an admin-edited one is left alone.
func TestUp_20261008045457_RelabelsInPlace(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := insertFourKVideoDownloaderTitle(t, db, fourKVideoDownloaderStaleBundleID)
	installerID := insertFourKVideoDownloaderInstaller(t, db, staleTitleID, 0, "storage-stale")

	generatedPolicyID := insertFourKVideoDownloaderPolicy(t, db, "[Install software] 4K Video Downloader",
		"SELECT 1 FROM apps WHERE bundle_identifier = 'com.4kdownload.ApplicationDirectories';", installerID)
	editedQuery := "SELECT 1 FROM apps WHERE bundle_identifier = 'com.4kdownload.ApplicationDirectories' AND path LIKE '/Applications/%';"
	editedPolicyID := insertFourKVideoDownloaderPolicy(t, db, "edited", editedQuery, installerID)
	execNoErr(t, db, `UPDATE policies SET updated_at = '2026-01-15 12:00:00' WHERE id = ?`, generatedPolicyID)

	applyNext(t, db)

	require.True(t, fourKVideoDownloaderTitleExists(t, db, staleTitleID))
	require.Equal(t, fourKVideoDownloaderBundleID, fourKVideoDownloaderTitleBundleID(t, db, staleTitleID))
	require.Equal(t, staleTitleID, fourKVideoDownloaderInstallerTitleID(t, db, installerID))

	require.Equal(t, "SELECT 1 FROM apps WHERE bundle_identifier = 'com.openmedia.4kvideodownloader';",
		fourKVideoDownloaderPolicyQuery(t, db, generatedPolicyID))
	require.Equal(t, editedQuery, fourKVideoDownloaderPolicyQuery(t, db, editedPolicyID))

	var updatedAt time.Time
	require.NoError(t, db.Get(&updatedAt, `SELECT updated_at FROM policies WHERE id = ?`, generatedPolicyID))
	require.Equal(t, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), updatedAt.UTC())
}

// Inventory already created the real title, so the stale one is merged into it: the
// installer, its history, its patch policy and its notification links move, and the stale
// title is dropped.
func TestUp_20261008045457_MergesIntoInventoryTitle(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := insertFourKVideoDownloaderTitle(t, db, fourKVideoDownloaderStaleBundleID)
	targetTitleID := insertFourKVideoDownloaderTitle(t, db, fourKVideoDownloaderBundleID)
	installerID := insertFourKVideoDownloaderInstaller(t, db, staleTitleID, 0, "storage-stale")

	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (hostname, osquery_host_id, node_key) VALUES ('h1', 'oh1', 'nk1')`)
	execNoErr(t, db, `
		INSERT INTO host_software_installs
			(host_id, execution_id, software_installer_id, software_title_id, install_script_exit_code)
		VALUES (?, 'exec-1', ?, ?, 0)`,
		hostID, installerID, staleTitleID)

	patchPolicyID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, query, description, checksum, team_id, type, patch_software_title_id)
		 VALUES ('4K Video Downloader up to date', 'SELECT 1', '', UNHEX(MD5('patch')), 0, 'patch', ?)`,
		staleTitleID)

	execNoErr(t, db, `INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, expires_at) VALUES ('n-1', ?, 'pending', 'patch', '{}', '2027-01-01 00:00:00')`, hostID)
	execNoErr(t, db,
		`INSERT INTO patch_notification_apps (notification_uuid, policy_id, software_title_id, software_installer_id) VALUES ('n-1', ?, ?, ?)`,
		patchPolicyID, staleTitleID, installerID)

	execNoErr(t, db,
		`UPDATE software_installers SET updated_at = '2026-01-15 12:00:00' WHERE id = ?`, installerID)

	applyNext(t, db)

	require.False(t, fourKVideoDownloaderTitleExists(t, db, staleTitleID))
	require.Equal(t, targetTitleID, fourKVideoDownloaderInstallerTitleID(t, db, installerID))

	var historyTitleID int64
	require.NoError(t, db.Get(&historyTitleID,
		`SELECT software_title_id FROM host_software_installs WHERE execution_id = 'exec-1'`))
	require.Equal(t, targetTitleID, historyTitleID)

	var patchTitleID int64
	require.NoError(t, db.Get(&patchTitleID, `SELECT patch_software_title_id FROM policies WHERE id = ?`, patchPolicyID))
	require.Equal(t, targetTitleID, patchTitleID)

	var notificationTitleID int64
	require.NoError(t, db.Get(&notificationTitleID,
		`SELECT software_title_id FROM patch_notification_apps WHERE notification_uuid = 'n-1'`))
	require.Equal(t, targetTitleID, notificationTitleID)

	var updatedAt time.Time
	require.NoError(t, db.Get(&updatedAt, `SELECT updated_at FROM software_installers WHERE id = ?`, installerID))
	require.Equal(t, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), updatedAt.UTC())
}

// A team that already has an installer and a patch policy on the target title keeps its
// stale installer and patch policy: moving them would put two installers on one title and
// violate the per-team patch policy key. The migration must not fail, and the stale title
// survives because they still depend on it.
func TestUp_20261008045457_SkipsTeamsWithExistingInstaller(t *testing.T) {
	db := applyUpToPrev(t)

	staleTitleID := insertFourKVideoDownloaderTitle(t, db, fourKVideoDownloaderStaleBundleID)
	targetTitleID := insertFourKVideoDownloaderTitle(t, db, fourKVideoDownloaderBundleID)
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1')`)

	insertFourKVideoDownloaderInstaller(t, db, targetTitleID, teamID, "storage-custom")
	blockedInstallerID := insertFourKVideoDownloaderInstaller(t, db, staleTitleID, teamID, "storage-fma-team")
	movableInstallerID := insertFourKVideoDownloaderInstaller(t, db, staleTitleID, 0, "storage-fma-global")

	execNoErr(t, db,
		`INSERT INTO policies (name, query, description, checksum, team_id, type, patch_software_title_id)
		 VALUES ('target patch', 'SELECT 1', '', UNHEX(MD5('target')), ?, 'patch', ?),
		        ('stale patch', 'SELECT 1', '', UNHEX(MD5('stale')), ?, 'patch', ?)`,
		teamID, targetTitleID, teamID, staleTitleID)

	applyNext(t, db)

	require.Equal(t, staleTitleID, fourKVideoDownloaderInstallerTitleID(t, db, blockedInstallerID))
	require.Equal(t, targetTitleID, fourKVideoDownloaderInstallerTitleID(t, db, movableInstallerID))
	require.True(t, fourKVideoDownloaderTitleExists(t, db, staleTitleID))

	var stalePatchTitleID int64
	require.NoError(t, db.Get(&stalePatchTitleID, `SELECT patch_software_title_id FROM policies WHERE name = 'stale patch'`))
	require.Equal(t, staleTitleID, stalePatchTitleID)
}

// Nothing to do when the FMA was never added.
func TestUp_20261008045457_NoOpWithoutStaleTitle(t *testing.T) {
	db := applyUpToPrev(t)

	targetTitleID := insertFourKVideoDownloaderTitle(t, db, fourKVideoDownloaderBundleID)

	applyNext(t, db)

	require.True(t, fourKVideoDownloaderTitleExists(t, db, targetTitleID))
	require.Equal(t, fourKVideoDownloaderBundleID, fourKVideoDownloaderTitleBundleID(t, db, targetTitleID))
}
