package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260702013056(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Table should exist and be empty.
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM setup_experience_software_installers`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Insert dependencies for a software_installer row.
	script := execNoErrLastID(t, db, `INSERT INTO script_contents (contents, md5_checksum) VALUES ('#!/bin/sh', 'abc')`)
	installerID := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(filename, extension, version, platform,
			 install_script_content_id, uninstall_script_content_id,
			 storage_id, package_ids, patch_query)
		VALUES ('hello.sh', 'sh', '', 'linux', ?, ?, 'stor-abc', '', '')
	`, script, script)

	// Insert a cross-platform selection row.
	_, err = db.Exec(`
		INSERT INTO setup_experience_software_installers
			(software_installer_id, platform, global_or_team_id)
		VALUES (?, 'darwin', 0)
	`, installerID)
	require.NoError(t, err)

	// Duplicate primary key should fail.
	_, err = db.Exec(`
		INSERT INTO setup_experience_software_installers
			(software_installer_id, platform, global_or_team_id)
		VALUES (?, 'darwin', 0)
	`, installerID)
	require.Error(t, err)

	// FK: referencing a non-existent installer should fail.
	_, err = db.Exec(`
		INSERT INTO setup_experience_software_installers
			(software_installer_id, platform, global_or_team_id)
		VALUES (99999, 'darwin', 0)
	`)
	require.Error(t, err)

	// ON DELETE CASCADE: deleting the installer removes the cross-platform row.
	_, err = db.Exec(`DELETE FROM software_installers WHERE id = ?`, installerID)
	require.NoError(t, err)

	err = db.QueryRow(`
		SELECT COUNT(*) FROM setup_experience_software_installers
		WHERE software_installer_id = ?
	`, installerID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count, "expected ON DELETE CASCADE to remove cross-platform selection row")
}
