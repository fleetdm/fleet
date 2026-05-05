package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250716152435(t *testing.T) {
	db := applyUpToPrev(t)

	// insert a software title
	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("Test App", "deb_packages", "")`)

	// insert script contents for install/uninstall
	scriptContentID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES ("md5", "echo 'Hello World'")`)

	// insert a software installer
	execNoErr(t, db, `
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
) VALUES (NULL, 0, ?, "a123b123", "foo.deb", "deb", "1.0.0", ?, ?, "linux", "")`, titleID, scriptContentID, scriptContentID)

	// Apply current migration.
	applyNext(t, db)

	// make sure column exists
	var blankCount int
	err := db.Get(&blankCount, `SELECT COUNT(*) FROM software_installers WHERE upgrade_code = ""`)
	require.NoError(t, err)
	require.Equal(t, 1, blankCount)
}
