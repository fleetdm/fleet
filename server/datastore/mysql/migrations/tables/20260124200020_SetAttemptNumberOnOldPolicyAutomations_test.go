package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260124200020(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a team
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ("Test Team")`)

	// Insert a policy
	policyID := execNoErrLastID(t, db, `
		INSERT INTO policies (name, query, description, team_id, checksum)
		VALUES ('test_policy', "SELECT 1", "", ?, "checksum")
	`, teamID)

	// Insert a script
	scriptContentID := execNoErrLastID(t, db, `
		INSERT INTO script_contents (md5_checksum, contents)
		VALUES ("md5hash", "echo test")
	`)
	scriptID := execNoErrLastID(t, db, `
		INSERT INTO scripts (team_id, global_or_team_id, name, script_content_id)
		VALUES (?, ?, "test.sh", ?)
	`, teamID, teamID, scriptContentID)

	// Insert a software title and installer
	titleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source)
		VALUES ("Test App", "apps")
	`)
	installerID := execNoErrLastID(t, db, `
		INSERT INTO software_installers (
			team_id, global_or_team_id, title_id, storage_id, filename,
			extension, version, install_script_content_id, uninstall_script_content_id, platform, package_ids
		) VALUES (?, ?, ?, "storage", "test.pkg", "pkg", "1.0", ?, ?, "darwin", "")
	`, teamID, teamID, titleID, scriptContentID, scriptContentID)

	// Insert host_script_results with NULL attempt_number and exit_code set
	scriptResultID1 := execNoErrLastID(t, db, `
		INSERT INTO host_script_results (
			host_id, execution_id, script_content_id, output, exit_code, script_id, policy_id, attempt_number
		) VALUES (1, "exec-1", ?, "output", 1, ?, ?, NULL)
	`, scriptContentID, scriptID, policyID)

	// Insert host_script_results with NULL attempt_number and exit_code NULL
	scriptResultID2 := execNoErrLastID(t, db, `
		INSERT INTO host_script_results (
			host_id, execution_id, script_content_id, output, exit_code, script_id, policy_id, attempt_number
		) VALUES (1, "exec-2", ?, "", NULL, ?, ?, NULL)
	`, scriptContentID, scriptID, policyID)

	// Insert host_software_installs with NULL attempt_number and install_script_exit_code set
	installID1 := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (
			execution_id, host_id, software_installer_id, user_id, self_service, policy_id,
			install_script_exit_code, attempt_number
		) VALUES ("install-1", 1, ?, NULL, 0, ?, 1, NULL)
	`, installerID, policyID)

	// Insert host_software_installs with NULL attempt_number and install_script_exit_code NULL
	installID2 := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (
			execution_id, host_id, software_installer_id, user_id, self_service, policy_id,
			install_script_exit_code, attempt_number
		) VALUES ("install-2", 1, ?, NULL, 0, ?, NULL, NULL)
	`, installerID, policyID)

	// Apply current migration
	applyNext(t, db)

	// Verify that completed executions were updated to attempt_number = 0
	type attemptNumber struct {
		AttemptNumber *int `db:"attempt_number"`
	}

	var scriptAttempt1 attemptNumber
	err := db.Get(&scriptAttempt1, `SELECT attempt_number FROM host_script_results WHERE id = ?`, scriptResultID1)
	require.NoError(t, err)
	require.NotNil(t, scriptAttempt1.AttemptNumber, "completed script should have attempt_number set")
	require.Equal(t, 0, *scriptAttempt1.AttemptNumber, "completed script should have attempt_number = 0")

	var scriptAttempt2 attemptNumber
	err = db.Get(&scriptAttempt2, `SELECT attempt_number FROM host_script_results WHERE id = ?`, scriptResultID2)
	require.NoError(t, err)
	require.Nil(t, scriptAttempt2.AttemptNumber, "pending script should still have attempt_number NULL")

	var installAttempt1 attemptNumber
	err = db.Get(&installAttempt1, `SELECT attempt_number FROM host_software_installs WHERE id = ?`, installID1)
	require.NoError(t, err)
	require.NotNil(t, installAttempt1.AttemptNumber, "completed install should have attempt_number set")
	require.Equal(t, 0, *installAttempt1.AttemptNumber, "completed install should have attempt_number = 0")

	var installAttempt2 attemptNumber
	err = db.Get(&installAttempt2, `SELECT attempt_number FROM host_software_installs WHERE id = ?`, installID2)
	require.NoError(t, err)
	require.Nil(t, installAttempt2.AttemptNumber, "pending install should still have attempt_number NULL")
}
