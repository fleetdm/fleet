package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20251226034530(t *testing.T) {
	cdHash1 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	db := applyUpToPrev(t)

	// Insert data to test the migration
	_, err := db.Exec(`
		INSERT INTO host_software_installed_paths (host_id, software_id, installed_path, team_identifier, executable_sha256)
		VALUES (1, 1, "/Applications/Fleet.app", "goteam", ?)
	`, cdHash1)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// confirm old column name no longer exists
	_, err = db.Exec(`SELECT executable_sha256 FROM host_software_installed_paths`)
	require.Error(t, err)

	var paths []fleet.HostSoftwareInstalledPath
	// binary_sha256 is left empty for old rows
	err = sqlx.Select(db, &paths, `
		SELECT 
			host_id,
			software_id,
			installed_path,
			team_identifier,
			cdhash_sha256,
			binary_sha256
		FROM host_software_installed_paths
	`)
	// confirms both new and updated column names are present
	require.NoError(t, err)
	require.Len(t, paths, 1)
	require.Equal(t, uint(1), paths[0].HostID)
	require.Equal(t, uint(1), paths[0].SoftwareID)
	require.Equal(t, "/Applications/Fleet.app", paths[0].InstalledPath)
	require.Equal(t, "goteam", paths[0].TeamIdentifier)
	require.Equal(t, cdHash1, *paths[0].CDHashSHA256)
	require.Nil(t, paths[0].BinarySHA256)

	cdHash2 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	binaryHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	_, err = db.Exec(`
		INSERT INTO host_software_installed_paths (host_id, software_id, installed_path, team_identifier, cdhash_sha256, binary_sha256)
		VALUES (2, 2, "/Applications/Go.app", "goteam", ?, ?)
	`, cdHash2, binaryHash)
	require.NoError(t, err)

	err = sqlx.Select(db, &paths, `
		SELECT 
			host_id,
			software_id,
			installed_path,
			team_identifier,
			cdhash_sha256,
			binary_sha256
		FROM host_software_installed_paths
	`)
	require.NoError(t, err)
	require.Len(t, paths, 2)

	// old row
	require.Equal(t, uint(1), paths[0].HostID)
	require.Equal(t, cdHash1, *paths[0].CDHashSHA256)
	// new row
	require.Equal(t, uint(2), paths[1].HostID)
	require.Equal(t, cdHash2, *paths[1].CDHashSHA256)
	require.Equal(t, binaryHash, *paths[1].BinarySHA256)
}
