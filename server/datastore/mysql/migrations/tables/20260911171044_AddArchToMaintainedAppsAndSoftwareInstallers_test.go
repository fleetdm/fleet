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
	armFMAID := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES ('Mozilla Firefox Nightly (ARM64)', 'firefox@nightly-arm64/windows', 'windows', 'Firefox Nightly')`)
	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES ('checksum', 'echo')`)

	insertInstaller := func(fma int64, filename, version, storage string, active bool) int64 {
		return execNoErrLastID(t, db, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, filename, version, platform, install_script_content_id,
				 uninstall_script_content_id, storage_id, package_ids, fleet_maintained_app_id, is_active, patch_query)
			VALUES (NULL, 0, ?, ?, ?, 'windows', ?, ?, ?, '', ?, ?, '')`,
			titleID, filename, version, scriptID, scriptID, storage, fma, active)
	}
	// Before this migration both builds can sit active on one title with nothing telling
	// them apart; the x64 build was added first.
	x64ActiveID := insertInstaller(fmaID, "nightly.msix", "158.0", "storage-x64-158", true)
	insertInstaller(fmaID, "nightly.msix", "157.0", "storage-x64-157", false)
	insertInstaller(armFMAID, "nightly-arm64.msix", "159.0", "storage-arm-159", true)

	// One pin backed by installers, one with nothing behind it.
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '^158')`, titleID)
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '1.0')`, otherTitleID)

	applyNext(t, db)

	// Applying again is a no-op, so a run that failed midway can be retried.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20260911171044(tx))
	require.NoError(t, tx.Commit())

	// Columns default to an empty architecture.
	var arch string
	require.NoError(t, db.QueryRow(`SELECT arch FROM fleet_maintained_apps WHERE id = ?`, fmaID).Scan(&arch))
	require.Empty(t, arch)
	require.NoError(t, db.QueryRow(`SELECT arch FROM software_installers WHERE id = ?`, x64ActiveID).Scan(&arch))
	require.Empty(t, arch)

	// The dedup key was replaced by one that includes arch.
	indexCount := func(name string) int {
		var count int
		require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'software_installers' AND index_name = ?`, name).Scan(&count))
		return count
	}
	require.Equal(t, 0, indexCount("idx_software_installers_dedup"))
	require.Equal(t, 1, indexCount("idx_software_installers_dedup_arch"))

	// The same FMA version can exist once per architecture.
	execNoErr(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, filename, version, platform, install_script_content_id,
			 uninstall_script_content_id, storage_id, package_ids, fleet_maintained_app_id, is_active, patch_query, arch)
		VALUES (NULL, 0, ?, 'nightly-arm64.msix', '158.0', 'windows', ?, ?, 'storage-arm-158', '', ?, 0, '', 'arm64')`,
		titleID, scriptID, scriptID, armFMAID)
	_, err = db.Exec(`
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, filename, version, platform, install_script_content_id,
			 uninstall_script_content_id, storage_id, package_ids, fleet_maintained_app_id, is_active, patch_query, arch)
		VALUES (NULL, 0, ?, 'dup.msix', '158.0', 'windows', ?, ?, 'storage-dup', '', ?, 0, '', '')`,
		titleID, scriptID, scriptID, fmaID)
	require.Error(t, err, "same team, title, arch, and version must still be rejected")

	// The backed pin followed the first-added FMA (x64), not the ARM64 sibling; the orphan is gone.
	var pinFMA int64
	var pinned string
	require.NoError(t, db.QueryRow(`SELECT fleet_maintained_app_id, pinned_version FROM software_title_team_pins WHERE team_id = 0 AND title_id = ?`, titleID).Scan(&pinFMA, &pinned))
	require.Equal(t, fmaID, pinFMA)
	require.Equal(t, "^158", pinned)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins`).Scan(&count))
	require.Equal(t, 1, count)

	// The ARM64 FMA on the same title gets its own pin, one per team, title, and FMA.
	execNoErr(t, db, `INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (0, ?, ?, '159.0')`, titleID, armFMAID)
	_, err = db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (0, ?, ?, '157.0')`, titleID, armFMAID)
	require.Error(t, err, "one pin per team, title, and FMA")
	_, err = db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version) VALUES (0, ?, 99999, '1.0')`, titleID)
	require.Error(t, err, "a pin must reference a catalog app")

	// Removing the FMA from the catalog removes its pin.
	execNoErr(t, db, `DELETE FROM fleet_maintained_apps WHERE id = ?`, armFMAID)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins WHERE fleet_maintained_app_id = ?`, armFMAID).Scan(&count))
	require.Equal(t, 0, count)
}
