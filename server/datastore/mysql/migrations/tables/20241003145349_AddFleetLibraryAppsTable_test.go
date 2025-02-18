package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241003145349(t *testing.T) {
	db := applyUpToPrev(t)

	// create an existing software installer before the migration
	execNoErr(t, db, `INSERT INTO script_contents (id, md5_checksum, contents) VALUES (1, 'checksum', 'script content')`)
	swiID := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id)
		VALUES
		(?,?,?,?,?,?,?)`, "sw1-installer.pkg", "1.2", "darwin", 1, "storage-id1", "", 1)

	// Apply current migration.
	applyNext(t, db)

	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM fleet_library_apps")
	require.NoError(t, err)
	require.Zero(t, count)

	// column was added and value is NULL
	var fleetLibraryAppID *uint
	err = db.Get(&fleetLibraryAppID, "SELECT fleet_library_app_id FROM software_installers WHERE id = ?", swiID)
	require.NoError(t, err)
	require.Nil(t, fleetLibraryAppID)
}
