package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260218165545(t *testing.T) {
	db := applyUpToPrev(t)

	// Test 1
	test1_macOSTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) 
		VALUES ('App 1', 'apps', 'com.example')
	`)
	test1_iosSoftwareID := execNoErrLastID(t, db, `
		INSERT INTO software (name, source, bundle_identifier, title_id, checksum)
		VALUES ('App 1', 'ios_apps', 'com.example', ?, ?)
	`, test1_macOSTitleID, []byte("App 1"))
	require.NotZero(t, test1_iosSoftwareID)

	// Test 2
	test2_macOSTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) 
		VALUES ('App 2', 'apps', 'com.example2')
	`)
	test2_iOSTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) 
		VALUES ('App 2', 'ios_apps', 'com.example2')
	`)
	test2_iosSoftwareID := execNoErrLastID(t, db, `
		INSERT INTO software (name, source, bundle_identifier, title_id, checksum)
		VALUES ('App 2', 'ios_apps', 'com.example2', ?, ?)
	`, test2_macOSTitleID, []byte("App 2"))
	require.NotZero(t, test2_iosSoftwareID)

	// Test 3
	test3_iOSTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) 
		VALUES ('App 3', 'ios_apps', 'com.example3')
	`)
	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('echo hello')), 'echo hello')`)
	test3_installerID := execNoErrLastID(t, db, `
	INSERT INTO software_installers (
		team_id,
		global_or_team_id,
		title_id,
		storage_id,
		filename,
		extension,
		version,
		install_script_content_id,
		uninstall_script_content_id,
		platform,
		package_ids
	) VALUES (NULL, 0, ?, "storage_id", "foo.pkg", "pkg", "1.0", ?, ?, "darwin", "")
	`, test3_iOSTitleID, scriptID, scriptID)
	require.NotZero(t, test3_installerID)

	// Apply current migration.
	applyNext(t, db)

	// Test 1
	var test1_newTitleID uint
	err := db.Get(&test1_newTitleID, `SELECT id FROM software_titles WHERE bundle_identifier = 'com.example' AND source = 'ios_apps'`)
	require.NoError(t, err)

	// iosSoftwareID should now be using the new software title
	var exists bool
	err = db.Get(&exists, `SELECT 1 FROM software WHERE id = ? AND title_id = ?`, test1_iosSoftwareID, test1_newTitleID)
	require.NoError(t, err)

	// Test 2
	// iosSoftwareID should now be using the new software title
	err = db.Get(&exists, `SELECT 1 FROM software WHERE id = ? AND title_id = ?`, test2_iosSoftwareID, test2_iOSTitleID)
	require.NoError(t, err)

	// Test 3
	var test3_newTitleID uint
	err = db.Get(&test3_newTitleID, `SELECT id FROM software_titles WHERE bundle_identifier = 'com.example3' AND source = 'apps'`)
	require.NoError(t, err)

	err = db.Get(&exists, `SELECT 1 FROM software_installers WHERE id = ? AND title_id = ?`, test3_installerID, test3_newTitleID)
	require.NoError(t, err)

}
