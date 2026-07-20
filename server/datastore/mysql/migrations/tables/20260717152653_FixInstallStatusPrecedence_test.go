package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260717152653(t *testing.T) {
	db := applyUpToPrev(t)

	// A failed install: the install script exited non-zero, but the post-install
	// script (which fleetd runs regardless) exited 0.
	const failedExecID = "failed-install-post-success"
	_, err := db.Exec(`
		INSERT INTO host_software_installs
			(execution_id, host_id, install_script_exit_code, post_install_script_exit_code)
		VALUES (?, 1, 1, 0)`, failedExecID)
	require.NoError(t, err)

	// A genuine success: both the install and post-install scripts exited 0.
	const successExecID = "install-and-post-success"
	_, err = db.Exec(`
		INSERT INTO host_software_installs
			(execution_id, host_id, install_script_exit_code, post_install_script_exit_code)
		VALUES (?, 1, 0, 0)`, successExecID)
	require.NoError(t, err)

	// Before the migration, the buggy precedence reports the failed install as installed.
	var before string
	require.NoError(t, db.QueryRow(`SELECT status FROM host_software_installs WHERE execution_id = ?`, failedExecID).Scan(&before))
	require.Equal(t, "installed", before)

	applyNext(t, db)

	// After the migration, a non-zero install-script exit code is terminal for both
	// generated columns, regardless of the post-install script result. The STORED
	// column is recomputed for the existing row by the table rebuild.
	assertStatus := func(execID, wantStatus, wantExecStatus string) {
		t.Helper()
		var status, execStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM host_software_installs WHERE execution_id = ?`, execID).Scan(&status))
		require.NoError(t, db.QueryRow(`SELECT execution_status FROM host_software_installs WHERE execution_id = ?`, execID).Scan(&execStatus))
		require.Equal(t, wantStatus, status)
		require.Equal(t, wantExecStatus, execStatus)
	}

	assertStatus(failedExecID, "failed_install", "failed_install")
	// Regression: a genuine success is still reported as installed.
	assertStatus(successExecID, "installed", "installed")

	// Regression: install succeeded but post-install failed is still a failure.
	const postFailedExecID = "install-success-post-failed"
	_, err = db.Exec(`
		INSERT INTO host_software_installs
			(execution_id, host_id, install_script_exit_code, post_install_script_exit_code)
		VALUES (?, 1, 0, 1)`, postFailedExecID)
	require.NoError(t, err)
	assertStatus(postFailedExecID, "failed_install", "failed_install")
}
