package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260629163945(t *testing.T) {
	db := applyUpToPrev(t)

	insertTitle := func(name string, source string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES (?, ?)`, name, source)
	}

	const installerInsert = `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, filename, extension, version, platform,
			 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query,
			 fleet_maintained_app_id, is_active)
		VALUES (?, ?, ?, ?, 'pkg', ?, ?, ?, ?, ?, '', '', ?, ?)`

	insertScript := func(seed string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO script_contents (contents, md5_checksum) VALUES ('#!/bin/sh', UNHEX(MD5(?)))`, seed)
	}

	// teamID nil means no-team with global_or_team_id 0, fmaID nil means a custom package.
	args := func(titleID int64, teamID *int64, platform string, version string, storage string, fmaID *int64, active int) []any {
		script := insertScript(storage + version)
		var globalOrTeamID int64
		if teamID != nil {
			globalOrTeamID = *teamID
		}
		return []any{teamID, globalOrTeamID, titleID, storage + "-" + version + ".pkg", version, platform, script, script, storage, fmaID, active}
	}
	insertInstaller := func(titleID int64, teamID *int64, platform string, version string, storage string, fmaID *int64, active int) int64 {
		return execNoErrLastID(t, db, installerInsert, args(titleID, teamID, platform, version, storage, fmaID, active)...)
	}
	tryInsertInstaller := func(titleID int64, teamID *int64, platform string, version string, storage string, fmaID *int64, active int) error {
		_, err := db.Exec(installerInsert, args(titleID, teamID, platform, version, storage, fmaID, active)...)
		return err
	}

	countRows := func(query string, qargs ...any) int {
		var n int
		require.NoError(t, db.QueryRow(query, qargs...).Scan(&n))
		return n
	}
	remainingIDs := func(titleID int64) []int64 {
		var ids []int64
		r, err := db.Query(`SELECT id FROM software_installers WHERE title_id = ? ORDER BY id`, titleID)
		require.NoError(t, err)
		for r.Next() {
			var id int64
			require.NoError(t, r.Scan(&id))
			ids = append(ids, id)
		}
		require.NoError(t, r.Err())
		require.NoError(t, r.Close())
		return ids
	}

	team := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('Team 1')`)
	label := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('L1', '')`)
	category := execNoErrLastID(t, db, `INSERT INTO software_categories (name) VALUES ('C1')`)
	fma := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES ('AppD', 'appd/darwin', 'darwin', 'com.appd')`)

	// Single custom package (no-team). Untouched.
	titleA := insertTitle("AppA", "apps")
	soloID := insertInstaller(titleA, nil, "darwin", "1.0", "hash-a", nil, 1)

	// Custom hash-duplicate (no-team): two active rows with the same content but
	// different versions, so the still-present version key allows seeding both. Each
	// carries a label and a category, and a policy points at the row to be deleted.
	titleB := insertTitle("AppB", "programs")
	keepB := insertInstaller(titleB, nil, "windows", "1.0", "hash-b", nil, 1)
	dupB := insertInstaller(titleB, nil, "windows", "2.0", "hash-b", nil, 1)
	for _, id := range []int64{keepB, dupB} {
		execNoErr(t, db, `INSERT INTO software_installer_labels (software_installer_id, label_id) VALUES (?, ?)`, id, label)
		execNoErr(t, db, `INSERT INTO software_installer_software_categories (software_installer_id, software_category_id) VALUES (?, ?)`, id, category)
	}
	policyID := execNoErrLastID(t, db, `
		INSERT INTO policies (name, query, description, checksum, software_installer_id)
		VALUES ('p1', 'SELECT 1', '', UNHEX(MD5('p1')), ?)`, dupB)

	// Custom hash-duplicate scoped to a team, different source.
	titleC := insertTitle("AppC", "rpm_packages")
	keepC := insertInstaller(titleC, &team, "linux", "1.0", "hash-c", nil, 1)
	dupC := insertInstaller(titleC, &team, "linux", "1.1", "hash-c", nil, 1)

	// FMA with the same bytes backing two versions. Tokens resolve to version, so both
	// must survive.
	titleD := insertTitle("AppD", "apps")
	fmaOld := insertInstaller(titleD, nil, "darwin", "1.0", "hash-d", &fma, 0)
	fmaActive := insertInstaller(titleD, nil, "darwin", "2.0", "hash-d", &fma, 1)

	applyNext(t, db)

	// The version key is gone, replaced by the dedup_token key.
	require.Zero(t, countRows(`
		SELECT COUNT(*) FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'software_installers'
			AND index_name = 'idx_software_installers_team_title_version'`))
	var dedupCols []string
	rows, err := db.Query(`
		SELECT column_name FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'software_installers'
			AND index_name = 'idx_software_installers_dedup'
		ORDER BY seq_in_index`)
	require.NoError(t, err)
	for rows.Next() {
		var col string
		require.NoError(t, rows.Scan(&col))
		dedupCols = append(dedupCols, col)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.Equal(t, []string{"global_or_team_id", "title_id", "dedup_token"}, dedupCols)

	// Single-package title untouched.
	require.Equal(t, []int64{soloID}, remainingIDs(titleA))

	// Custom hash-duplicates collapse to the first-added row, which stays active.
	require.Equal(t, []int64{keepB}, remainingIDs(titleB))
	require.Equal(t, []int64{keepC}, remainingIDs(titleC))
	require.NotContains(t, remainingIDs(titleC), dupC)
	require.Equal(t, 1, countRows(`SELECT is_active FROM software_installers WHERE id = ?`, keepB))
	require.Equal(t, 1, countRows(`SELECT is_active FROM software_installers WHERE id = ?`, keepC))

	// The survivor keeps its label and category. The deleted row's cascade away.
	require.Equal(t, 1, countRows(`SELECT COUNT(*) FROM software_installer_labels WHERE software_installer_id = ?`, keepB))
	require.Equal(t, 1, countRows(`SELECT COUNT(*) FROM software_installer_software_categories WHERE software_installer_id = ?`, keepB))
	require.Zero(t, countRows(`SELECT COUNT(*) FROM software_installer_labels WHERE software_installer_id = ?`, dupB))
	require.Zero(t, countRows(`SELECT COUNT(*) FROM software_installer_software_categories WHERE software_installer_id = ?`, dupB))

	// The policy was re-pointed off the deleted row onto the survivor.
	var repointed int64
	require.NoError(t, db.QueryRow(`SELECT software_installer_id FROM policies WHERE id = ?`, policyID).Scan(&repointed))
	require.Equal(t, keepB, repointed)

	// FMA same-hash-different-version rows both survive.
	require.Equal(t, []int64{fmaOld, fmaActive}, remainingIDs(titleD))

	// New key behavior. Custom same-version-different-hash is accepted, and these could not
	// be seeded before the migration because the old version key blocked two rows sharing a
	// version.
	titleE := insertTitle("AppE", "apps")
	require.NoError(t, tryInsertInstaller(titleE, nil, "darwin", "9.0", "hash-e1", nil, 1))
	require.NoError(t, tryInsertInstaller(titleE, nil, "darwin", "9.0", "hash-e2", nil, 1))

	// A second package with identical bytes on the same title is rejected by the key.
	require.Error(t, tryInsertInstaller(titleE, nil, "darwin", "8.0", "hash-e1", nil, 1))

	// FMA can still cache another version backed by the same bytes.
	require.NoError(t, tryInsertInstaller(titleD, nil, "darwin", "3.0", "hash-d", &fma, 0))
}
