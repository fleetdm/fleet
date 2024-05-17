package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240517135955(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...

	script1 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo hi', 'a')")
	script2 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo bye', 'b')")

	software := execNoErrLastID(t, db, `
INSERT INTO software_installers (
  filename,
  version,
  platform,
  install_script_content_id,
  post_install_script_content_id,
  storage_id
) VALUES (
  'fleet',
  '1.0.0',
  'windows',
  ?,
  ?,
  'a'
)
`, script1, script2)

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...

	var self_service bool
	err := db.Get(&self_service, "SELECT self_service FROM software_installers WHERE id = ?", software)
	require.NoError(t, err)
	require.False(t, self_service)
}
