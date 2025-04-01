package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250331154206(t *testing.T) {
	db := applyUpToPrev(t)

	// this is basically the same test as 20241021224359_AddExecutionStatusToHostSoftwareInstalls_test.go
	// to ensure that the same state still corresponds to the same resulting statuses, after migration
	// and change of the status and execution_status columns.
	hostID := insertHost(t, db, nil)
	dataStmts := `
	  INSERT INTO script_contents (id, md5_checksum, contents) VALUES
	    (1, 'checksum', 'script content');

	  INSERT INTO software_titles (id, name, source, browser) VALUES  (1, 'Foo.app', 'apps', '');

	  INSERT INTO software_installers
	    (id, title_id, filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id)
	  VALUES
	    (1, 1, 'foo-installer.pkg', '1.1', 'darwin', 1, 'storage-id', '', 1);
	`
	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	hsiStmt := `
	INSERT INTO host_software_installs (
		host_id,
		execution_id,
		software_installer_id,
		install_script_exit_code,
		uninstall_script_exit_code,
		updated_at,
		uninstall,
		removed
	) VALUES (?, ?, ?, ?, ?, '2024-10-01 00:00:00', ?, 1)`
	hsiInstall := execNoErrLastID(t, db, hsiStmt, hostID, "execution-id1", 1, 0, nil, 0)
	hsiUninstall := execNoErrLastID(t, db, hsiStmt, hostID, "execution-id2", 1, nil, 0, 1)

	// Apply current migration.
	applyNext(t, db)

	var statuses struct {
		Status          *string `db:"status"`
		ExecutionStatus *string `db:"execution_status"`
	}

	err = db.Get(&statuses, "SELECT status, execution_status FROM host_software_installs WHERE id = ?", hsiInstall)
	require.NoError(t, err)
	require.NotNil(t, statuses.ExecutionStatus)
	require.Equal(t, "installed", *statuses.ExecutionStatus)
	require.Nil(t, statuses.Status)

	err = db.Get(&statuses, "SELECT status, execution_status FROM host_software_installs WHERE id = ?", hsiUninstall)
	require.NoError(t, err)
	require.Nil(t, statuses.ExecutionStatus) // uninstalls have null status
	require.Nil(t, statuses.Status)

	execNoErr(t, db, `UPDATE host_software_installs SET removed = 0`)

	err = db.Get(&statuses, "SELECT status, execution_status FROM host_software_installs WHERE id = ?", hsiInstall)
	require.NoError(t, err)
	require.NotNil(t, statuses.ExecutionStatus)
	require.Equal(t, "installed", *statuses.ExecutionStatus)
	require.NotNil(t, statuses.Status)
	require.Equal(t, "installed", *statuses.Status)

	err = db.Get(&statuses, "SELECT status, execution_status FROM host_software_installs WHERE id = ?", hsiUninstall)
	require.NoError(t, err)
	require.Nil(t, statuses.ExecutionStatus) // uninstalls have null status
	require.Nil(t, statuses.Status)

	execNoErr(t, db, `UPDATE host_software_installs SET canceled = 1`)

	err = db.Get(&statuses, "SELECT status, execution_status FROM host_software_installs WHERE id = ?", hsiInstall)
	require.NoError(t, err)
	require.NotNil(t, statuses.ExecutionStatus)
	require.Equal(t, "canceled_install", *statuses.ExecutionStatus)
	require.NotNil(t, statuses.Status)
	require.Equal(t, "canceled_install", *statuses.Status)

	err = db.Get(&statuses, "SELECT status, execution_status FROM host_software_installs WHERE id = ?", hsiUninstall)
	require.NoError(t, err)
	require.NotNil(t, statuses.ExecutionStatus)
	require.Equal(t, "canceled_uninstall", *statuses.ExecutionStatus)
	require.NotNil(t, statuses.Status)
	require.Equal(t, "canceled_uninstall", *statuses.Status)
}
