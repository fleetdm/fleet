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

	// Apply current migration.
	applyNext(t, db)

	// The column collation is case insensitive, so read back with a binary
	// collation to see the stored casing rather than a case-folded match.
	extensionOf := func(id int64) string {
		var got string
		err := db.QueryRow(`SELECT extension COLLATE utf8mb4_bin FROM software_installers WHERE id = ?`, id).Scan(&got)
		require.NoError(t, err)
		return got
	}

	require.Equal(t, "exe", extensionOf(upperExe))
	require.Equal(t, "dmg", extensionOf(upperDmg))
	require.Equal(t, "exe", extensionOf(alreadyLower))
	require.Equal(t, "tar.gz", extensionOf(tarball))
}
