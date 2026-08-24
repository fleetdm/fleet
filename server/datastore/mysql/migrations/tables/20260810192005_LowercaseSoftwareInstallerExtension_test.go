package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260810192005(t *testing.T) {
	db := applyUpToPrev(t)

	insertInstaller := func(filename string, extension string, storage string) int64 {
		titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES (?, 'programs')`, filename)
		scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (contents, md5_checksum) VALUES ('#!/bin/sh', UNHEX(MD5(?)))`, storage)
		return execNoErrLastID(t, db, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, filename, extension, version, platform,
				 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
			VALUES (NULL, 0, ?, ?, ?, '1.0', 'windows', ?, ?, ?, '', '')`,
			titleID, filename, extension, scriptID, scriptID, storage)
	}

	upperExe := insertInstaller("BANDIVIEW-SETUP-X64.EXE", "EXE", "storage-upper-exe")
	upperDmg := insertInstaller("Joplin-arm64.DMG", "DMG", "storage-upper-dmg")
	alreadyLower := insertInstaller("setup.exe", "exe", "storage-lower-exe")
	tarball := insertInstaller("package.tar.gz", "tar.gz", "storage-targz")

	// The column collation is case insensitive, so read the extension back with a
	// binary collation to see the stored casing rather than a case-folded match.
	type installerRow struct {
		ID        int64  `db:"id"`
		Extension string `db:"extension"`
		UpdatedAt string `db:"updated_at"`
	}
	snapshot := func() map[int64]installerRow {
		var installers []installerRow
		err := db.Select(&installers,
			`SELECT id, extension COLLATE utf8mb4_bin AS extension, updated_at FROM software_installers`)
		require.NoError(t, err)
		byID := make(map[int64]installerRow, len(installers))
		for _, installer := range installers {
			byID[installer.ID] = installer
		}
		return byID
	}
	before := snapshot()

	// Apply current migration.
	applyNext(t, db)

	after := snapshot()

	require.Equal(t, "exe", after[upperExe].Extension)
	require.Equal(t, "dmg", after[upperDmg].Extension)
	require.Equal(t, "exe", after[alreadyLower].Extension)
	require.Equal(t, "tar.gz", after[tarball].Extension)

	// updated_at is ON UPDATE CURRENT_TIMESTAMP, and MySQL skips the write for a
	// row whose value doesn't change, so only the rewritten rows get a new one.
	require.NotEqual(t, before[upperExe].UpdatedAt, after[upperExe].UpdatedAt)
	require.NotEqual(t, before[upperDmg].UpdatedAt, after[upperDmg].UpdatedAt)
	require.Equal(t, before[alreadyLower].UpdatedAt, after[alreadyLower].UpdatedAt)
	require.Equal(t, before[tarball].UpdatedAt, after[tarball].UpdatedAt)
}
