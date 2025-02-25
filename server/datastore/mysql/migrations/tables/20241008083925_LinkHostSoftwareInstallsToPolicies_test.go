package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241008083925(t *testing.T) {
	db := applyUpToPrev(t)

	// insert a team
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ("Foo")`)

	// insert a policy
	policyID := execNoErrLastID(t, db, `INSERT INTO policies (name, query, description, team_id, checksum)
		VALUES ('test_policy', "SELECT 1", "", ?, "a123b123")`, teamID)

	// insert a software title
	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("Test App", "deb_packages", "")`)

	// insert script contents for install/uninstall
	scriptContentID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES ("md5", "echo 'Hello World'")`)

	// insert a software installer
	installerID := execNoErrLastID(t, db, `
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

	// insert a software install result
	hostSoftwareInstallID := execNoErrLastID(t, db, `INSERT INTO host_software_installs
		    (execution_id, host_id, software_installer_id, user_id, self_service, policy_id)
		  VALUES ("a123b123", 1337, ?, NULL, 0, ?)`, installerID, policyID)

	// delete the associated policy
	execNoErr(t, db, `DELETE FROM policies WHERE id = ?`, policyID)

	// policy ID should be null but install result should still exist
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM host_software_installs WHERE policy_id IS NULL AND id = ?", hostSoftwareInstallID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
