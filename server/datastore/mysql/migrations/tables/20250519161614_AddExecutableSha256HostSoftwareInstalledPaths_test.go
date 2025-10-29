package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20250519161614(t *testing.T) {
	db := applyUpToPrev(t)

	// create an existing host_mdm_actions row
	_, err := db.Exec(`
		INSERT INTO host_software_installed_paths (host_id, software_id, installed_path, team_identifier)
		VALUES (1, 1, "/Applications/Fleet.app", "goteam")
	`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	var hostSoftwareInstalledPaths []struct {
		HostID           uint    `db:"host_id"`
		SoftwareID       uint    `db:"software_id"`
		InstalledPath    string  `db:"installed_path"`
		TeamIdentifier   string  `db:"team_identifier"`
		ExecutableSha256 *string `db:"executable_sha256"`
	}

	// executable_sha256 is left empty for old rows
	err = sqlx.Select(db, &hostSoftwareInstalledPaths, `
		SELECT 
			host_id,
			software_id,
			installed_path,
			team_identifier,
			executable_sha256
		FROM host_software_installed_paths
	`)
	require.NoError(t, err)
	require.Len(t, hostSoftwareInstalledPaths, 1)
	require.Equal(t, uint(1), hostSoftwareInstalledPaths[0].HostID)
	require.Equal(t, uint(1), hostSoftwareInstalledPaths[0].SoftwareID)
	require.Equal(t, "/Applications/Fleet.app", hostSoftwareInstalledPaths[0].InstalledPath)
	require.Equal(t, "goteam", hostSoftwareInstalledPaths[0].TeamIdentifier)
	require.Nil(t, hostSoftwareInstalledPaths[0].ExecutableSha256)

	_, err = db.Exec(`
		INSERT INTO host_software_installed_paths (host_id, software_id, installed_path, team_identifier, executable_sha256)
		VALUES (1, 2, "/Applications/Go.app", "goteam", "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
	`)
	require.NoError(t, err)

	err = sqlx.Select(db, &hostSoftwareInstalledPaths, `
		SELECT 
			host_id,
			software_id,
			installed_path,
			team_identifier,
			executable_sha256
		FROM host_software_installed_paths
	`)
	require.NoError(t, err)
	require.Len(t, hostSoftwareInstalledPaths, 2)
	require.Equal(t, uint(1), hostSoftwareInstalledPaths[1].HostID)
	require.Equal(t, uint(2), hostSoftwareInstalledPaths[1].SoftwareID)
	require.Equal(t, "/Applications/Go.app", hostSoftwareInstalledPaths[1].InstalledPath)
	require.Equal(t, "goteam", hostSoftwareInstalledPaths[1].TeamIdentifier)
	require.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", *hostSoftwareInstalledPaths[1].ExecutableSha256)
}
