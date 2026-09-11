package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260911171044(t *testing.T) {
	db := applyUpToPrev(t)

	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, extension_for, upgrade_code) VALUES ('Firefox Nightly', 'programs', '', '')`)
	otherTitleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, extension_for, upgrade_code) VALUES ('Orphan', 'programs', '', '')`)
	fmaID := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES ('Mozilla Firefox Nightly', 'firefox@nightly/windows', 'windows', 'Firefox Nightly')`)
	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES ('checksum', 'echo')`)

	insertInstaller := func(version string, active bool) int64 {
		return execNoErrLastID(t, db, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, filename, version, platform, install_script_content_id,
				 uninstall_script_content_id, storage_id, package_ids, fleet_maintained_app_id, is_active, patch_query)
			VALUES (NULL, 0, ?, 'nightly.msix', ?, 'windows', ?, ?, ?, '', ?, ?, '')`,
			titleID, version, scriptID, scriptID, "storage-"+version, fmaID, active)
	}
	activeID := insertInstaller("158.0", true)
	insertInstaller("157.0", false)

	// One pin backed by an active installer, one with nothing behind it.
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '^158')`, titleID)
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '1.0')`, otherTitleID)

	applyNext(t, db)

	// Columns default to an empty architecture.
	var arch string
	require.NoError(t, db.QueryRow(`SELECT arch FROM fleet_maintained_apps WHERE id = ?`, fmaID).Scan(&arch))
	require.Empty(t, arch)
	require.NoError(t, db.QueryRow(`SELECT arch FROM software_installers WHERE id = ?`, activeID).Scan(&arch))
	require.Empty(t, arch)

	// The dedup key now includes arch: the same FMA version can exist once per architecture.
	execNoErr(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, filename, version, platform, install_script_content_id,
			 uninstall_script_content_id, storage_id, package_ids, fleet_maintained_app_id, is_active, patch_query, arch)
		VALUES (NULL, 0, ?, 'nightly-arm64.msix', '158.0', 'windows', ?, ?, 'storage-arm64', '', ?, 1, '', 'arm64')`,
		titleID, scriptID, scriptID, fmaID)
	_, err := db.Exec(`
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, filename, version, platform, install_script_content_id,
			 uninstall_script_content_id, storage_id, package_ids, fleet_maintained_app_id, is_active, patch_query, arch)
		VALUES (NULL, 0, ?, 'dup.msix', '158.0', 'windows', ?, ?, 'storage-dup', '', ?, 0, '', '')`,
		titleID, scriptID, scriptID, fmaID)
	require.Error(t, err, "same team, title, arch, and version must still be rejected")

	// The backed pin is now scoped to its FMA; the orphan is gone.
	var pinFMA int64
	var pinned string
	require.NoError(t, db.QueryRow(`SELECT fleet_maintained_app_id, pinned_version FROM software_title_team_pins WHERE team_id = 0 AND title_id = ?`, titleID).Scan(&pinFMA, &pinned))
	require.Equal(t, fmaID, pinFMA)
	require.Equal(t, "^158", pinned)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins`).Scan(&count))
	require.Equal(t, 1, count)

	// A second FMA on the same title gets its own pin.
	otherFMAID := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier, arch) VALUES ('Mozilla Firefox Nightly (ARM64)', 'firefox@nightly-arm64/windows', 'windows', 'Firefox Nightly', 'arm64')`)
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (0, ?, ?, '157.0')`, titleID, otherFMAID)
	_, err = db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (0, ?, ?, '156.0')`, titleID, otherFMAID)
	require.Error(t, err, "one pin per team, title, and FMA")

	// Removing the FMA from the catalog removes its pin.
	execNoErr(t, db, `DELETE FROM fleet_maintained_apps WHERE id = ?`, otherFMAID)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins WHERE fleet_maintained_app_id = ?`, otherFMAID).Scan(&count))
	require.Equal(t, 0, count)
}
