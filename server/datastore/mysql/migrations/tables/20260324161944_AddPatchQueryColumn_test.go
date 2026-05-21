package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260324161944(t *testing.T) {
	db := applyUpToPrev(t)

	insertInstallerStmt := `
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
	) VALUES (NULL, 0, ?, "storage_id", ?, "pkg", "1.0", ?, ?, "darwin", "")
`

	insertTitleStmt := `
	INSERT INTO software_titles (name, source, bundle_identifier) 
	VALUES (?, 'apps', ?)
`

	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('echo hello')), 'echo hello')`)

	title1 := execNoErrLastID(t, db, insertTitleStmt, "App 1", "com.app1")
	title2 := execNoErrLastID(t, db, insertTitleStmt, "App 2", "com.app2")
	installer1 := execNoErrLastID(t, db, insertInstallerStmt, title1, "app1.pkg", scriptID, scriptID)
	installer2 := execNoErrLastID(t, db, insertInstallerStmt, title2, "app2.pkg", scriptID, scriptID)

	var timestamp, timestamp2 time.Time
	require.NoError(t, db.Get(&timestamp, `SELECT updated_at FROM software_installers WHERE id = ?`, installer1))

	// Apply current migration.
	applyNext(t, db)

	var patchQuery string
	require.NoError(t, db.Get(&patchQuery, `SELECT patch_query FROM software_installers WHERE id = ?`, installer1))
	require.Equal(t, "", patchQuery)
	require.NoError(t, db.Get(&patchQuery, `SELECT patch_query FROM software_installers WHERE id = ?`, installer2))
	require.Equal(t, "", patchQuery)
	require.NoError(t, db.Get(&timestamp2, `SELECT updated_at FROM software_installers WHERE id = ?`, installer1))
	require.Equal(t, timestamp, timestamp2)

}
