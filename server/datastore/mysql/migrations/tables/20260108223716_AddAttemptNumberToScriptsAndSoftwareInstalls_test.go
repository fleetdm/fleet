package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260108223716(t *testing.T) {
	db := applyUpToPrev(t)

	// insert a team
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ("Team 1")`)

	// insert a policy
	policyID := execNoErrLastID(t, db, `
		INSERT INTO policies (name, query, description, team_id, checksum)
		VALUES ('test_policy', "SELECT 1", "", ?, "checksum")
	`, teamID)

	// insert a software title
	titleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source)
		VALUES ("Test App", "apps")
	`)

	// insert script contents for install/uninstall
	scriptContentID := execNoErrLastID(t, db, `
		INSERT INTO script_contents (md5_checksum, contents)
		VALUES ("md5", "echo 'installing'")
	`)

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
		) VALUES (?, ?, ?, "storageid", "testapp.pkg", "pkg", "1.0.0", ?, ?, "macos", "")
	`, teamID, teamID, titleID, scriptContentID, scriptContentID)

	// insert a software install result
	hostSoftwareInstallID := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, user_id, self_service, policy_id)
		VALUES ("exection 1", 1, ?, NULL, 0, ?)
	`, installerID, policyID)

	// insert a script
	scriptID := execNoErrLastID(t, db, `
		INSERT INTO scripts (team_id, global_or_team_id, name, script_content_id)
		VALUES (?, ?, "test_script.sh", ?)
	`, teamID, teamID, scriptContentID)

	// insert a script result
	hostScriptResultID := execNoErrLastID(t, db, `
		INSERT INTO host_script_results (host_id, execution_id, script_content_id, output, exit_code, script_id, policy_id)
		VALUES (1, "exec-script-1", ?, "output", 1, ?, ?)
	`, scriptContentID, scriptID, policyID)

	// Apply current migration.
	applyNext(t, db)

	var attempt_number []sql.NullInt64
	err := db.Select(&attempt_number,
		`SELECT attempt_number FROM host_software_installs WHERE id = ?`, hostSoftwareInstallID)
	require.NoError(t, err)
	// check that the default is NULL
	require.Equal(t, sql.NullInt64{Valid: false}, attempt_number[0])

	// insert another software install result attempt
	hostSoftwareInstallID = execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, user_id, self_service, policy_id, attempt_number)
		VALUES ("exection 2", 1, ?, NULL, 0, ?, 1)
	`, installerID, policyID)

	err = db.Select(&attempt_number,
		`SELECT attempt_number FROM host_software_installs WHERE id = ?`, hostSoftwareInstallID)
	require.NoError(t, err)
	require.Equal(t, sql.NullInt64{Valid: true, Int64: 1}, attempt_number[0])

	var scriptAttemptNumber []sql.NullInt64
	err = db.Select(&scriptAttemptNumber,
		`SELECT attempt_number FROM host_script_results WHERE id = ?`, hostScriptResultID)
	require.NoError(t, err)
	require.Equal(t, sql.NullInt64{Valid: false}, scriptAttemptNumber[0])

	// insert another script result with attempt_number set
	hostScriptResultID2 := execNoErrLastID(t, db, `
		INSERT INTO host_script_results (host_id, execution_id, script_content_id, output, exit_code, script_id, policy_id, attempt_number)
		VALUES (1, "exec-script-2", ?, "output", 0, ?, ?, 2)
	`, scriptContentID, scriptID, policyID)

	err = db.Select(&scriptAttemptNumber,
		`SELECT attempt_number FROM host_script_results WHERE id = ?`, hostScriptResultID2)
	require.NoError(t, err)
	require.Equal(t, sql.NullInt64{Valid: true, Int64: 2}, scriptAttemptNumber[0])
}
