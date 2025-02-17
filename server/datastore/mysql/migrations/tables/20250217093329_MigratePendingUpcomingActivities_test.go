package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20250217093329_None(t *testing.T) {
	db := applyUpToPrev(t)
	// Apply current migration.
	applyNext(t, db)
	assertRowCount(t, db, "upcoming_activities", 0)
}

func TestUp_20250217093329_Script(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)

	// create a script content
	csum := md5ChecksumScriptContent(`echo 1`)
	scriptContentID := execNoErrLastID(t, db, `INSERT INTO script_contents
		(md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum, `echo 1`)

	// insert a couple pending but one has host_deleted_at set, and a non-pending script
	execIDPending, execIDDeleted, execIDDone := uuid.NewString(), uuid.NewString(), uuid.NewString()
	db.Exec(`INSERT INTO host_script_results
		(host_id, execution_id, output, script_content_id, host_deleted_at)
	VALUES (?, ?, '', ?, ?)`, hostID, execIDPending, scriptContentID, nil)
	db.Exec(`INSERT INTO host_script_results
		(host_id, execution_id, output, script_content_id, host_deleted_at)
	VALUES (?, ?, '', ?, ?)`, hostID, execIDDeleted, scriptContentID, time.Now())
	db.Exec(`INSERT INTO host_script_results
		(host_id, execution_id, output, script_content_id, exit_code)
	VALUES (?, ?, '', ?, 0)`, hostID, execIDDone, scriptContentID)

	// Apply current migration.
	applyNext(t, db)
	assertRowCount(t, db, "upcoming_activities", 1)
	assertRowCount(t, db, "script_upcoming_activities", 1)
	assertRowCount(t, db, "software_install_upcoming_activities", 0)
	assertRowCount(t, db, "vpp_app_upcoming_activities", 0)
	var execID string
	err := db.Get(&execID, `SELECT execution_id FROM upcoming_activities`)
	require.NoError(t, err)
	require.Equal(t, execIDPending, execID)
}

func TestUp_20250217093329_SoftwareInstall(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)

	script := `echo 1`
	csum := md5ChecksumScriptContent(script)
	scriptContentID := execNoErrLastID(t, db, `INSERT INTO script_contents
		(md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum, script)
	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles
		(name, source, browser) VALUES ('Foo.app', 'apps', '')`)
	installerID := execNoErrLastID(t, db, `INSERT INTO software_installers
		(title_id, filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id)
		VALUES (?, 'foo.pkg', '1.1', 'darwin', ?, 'storage-id', '', ?)`, titleID, scriptContentID, scriptContentID)

	// insert a few pending but one has host_deleted_at, uninstall or removed set, and a non-pending install
	hsiStmt := `
	INSERT INTO host_software_installs (
		host_id, execution_id, software_installer_id, install_script_exit_code,
		host_deleted_at, removed, uninstall
	) VALUES (?, ?, ?, ?, ?, ?, ?)`
	execIDPending, execIDDeleted, execIDUninstall, execIDRemoved, execIDFailed :=
		uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	execNoErr(t, db, hsiStmt, hostID, execIDPending, installerID, nil, nil, false, false)
	execNoErr(t, db, hsiStmt, hostID, execIDDeleted, installerID, nil, time.Now(), false, false)
	execNoErr(t, db, hsiStmt, hostID, execIDUninstall, installerID, nil, nil, false, true)
	execNoErr(t, db, hsiStmt, hostID, execIDRemoved, installerID, nil, nil, true, false)
	execNoErr(t, db, hsiStmt, hostID, execIDFailed, installerID, 1, nil, false, false)

	t.Log("exec IDs: ", execIDPending, execIDDeleted, execIDUninstall, execIDRemoved, execIDFailed)

	applyNext(t, db)
	assertRowCount(t, db, "upcoming_activities", 2)
	assertRowCount(t, db, "software_install_upcoming_activities", 2)
	assertRowCount(t, db, "vpp_app_upcoming_activities", 0)
	assertRowCount(t, db, "script_upcoming_activities", 0)
	var execIDs []string
	err := db.Select(&execIDs, `SELECT execution_id FROM upcoming_activities`)
	require.NoError(t, err)
	// will add both the pending install and uninstall to upcoming, but not the
	// host deleted entry, the removed and the failed install
	require.ElementsMatch(t, []string{execIDPending, execIDUninstall}, execIDs)
}

func TestUp_20250217093329_SoftwareUninstall(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)

	script := `echo 1`
	csum := md5ChecksumScriptContent(script)
	scriptContentID := execNoErrLastID(t, db, `INSERT INTO script_contents
		(md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum, script)
	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles
		(name, source, browser) VALUES ('Foo.app', 'apps', '')`)
	installerID := execNoErrLastID(t, db, `INSERT INTO software_installers
		(title_id, filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id)
		VALUES (?, 'foo.pkg', '1.1', 'darwin', ?, 'storage-id', '', ?)`, titleID, scriptContentID, scriptContentID)

	// insert a few pending but one has host_deleted_at, is an install or has
	// removed set, and a non-pending uninstall
	hsiStmt := `
	INSERT INTO host_software_installs (
		host_id, execution_id, software_installer_id, uninstall_script_exit_code,
		host_deleted_at, removed, uninstall
	) VALUES (?, ?, ?, ?, ?, ?, ?)`
	execIDPending, execIDDeleted, execIDInstall, execIDRemoved, execIDFailed :=
		uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	execNoErr(t, db, hsiStmt, hostID, execIDPending, installerID, nil, nil, false, true)
	execNoErr(t, db, hsiStmt, hostID, execIDDeleted, installerID, nil, time.Now(), false, true)
	execNoErr(t, db, hsiStmt, hostID, execIDInstall, installerID, nil, nil, false, false)
	execNoErr(t, db, hsiStmt, hostID, execIDRemoved, installerID, nil, nil, true, true)
	execNoErr(t, db, hsiStmt, hostID, execIDFailed, installerID, 1, nil, false, true)

	t.Log("exec IDs: ", execIDPending, execIDDeleted, execIDInstall, execIDRemoved, execIDFailed)

	applyNext(t, db)
	assertRowCount(t, db, "upcoming_activities", 2)
	assertRowCount(t, db, "software_install_upcoming_activities", 2)
	assertRowCount(t, db, "vpp_app_upcoming_activities", 0)
	assertRowCount(t, db, "script_upcoming_activities", 0)
	var execIDs []string
	err := db.Select(&execIDs, `SELECT execution_id FROM upcoming_activities`)
	require.NoError(t, err)
	// will add both the pending install and uninstall to upcoming, but not the
	// host deleted entry, the removed and the failed uninstall
	require.ElementsMatch(t, []string{execIDPending, execIDInstall}, execIDs)
}

func TestUp_20250217093329_VPPInstall(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)
	hostUUID := "12345678-1234-1234-1234-123456789012"

	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles
		(name, source, browser) VALUES ('Foo.app', 'apps', '')`)
	adamID := "abcd"
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id) 
		VALUES (?,?,?)`, adamID, "darwin", titleID)

	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) 
		VALUES (?, ?)`, hostUUID, "auth")
	execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`, hostUUID, hostUUID, "device", "topic", "magic", "hex", time.Now())

	// create a few pending but one is removed, and a non-pending install
	execIDPending, execIDRemoved, execIDDone := uuid.NewString(), uuid.NewString(), uuid.NewString()
	execNoErr(t, db, `INSERT INTO host_vpp_software_installs 
		(host_id, adam_id, platform, command_uuid, removed) VALUES (?, ?, ?, ?, ?)`,
		hostID, adamID, "darwin", execIDPending, false)
	execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`,
		execIDPending, "InstallApplication", "<?xml")
	execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`,
		hostUUID, execIDPending)

	execNoErr(t, db, `INSERT INTO host_vpp_software_installs 
		(host_id, adam_id, platform, command_uuid, removed) VALUES (?, ?, ?, ?, ?)`,
		hostID, adamID, "darwin", execIDRemoved, true)
	execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`,
		execIDRemoved, "InstallApplication", "<?xml")
	execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`,
		hostUUID, execIDRemoved)

	execNoErr(t, db, `INSERT INTO host_vpp_software_installs 
		(host_id, adam_id, platform, command_uuid, removed) VALUES (?, ?, ?, ?, ?)`,
		hostID, adamID, "darwin", execIDDone, false)
	execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`,
		execIDDone, "InstallApplication", "<?xml")
	execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`,
		hostUUID, execIDDone)
	execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, ?)`,
		hostUUID, execIDDone, "Acknowledged", "<?xml")

	applyNext(t, db)
	assertRowCount(t, db, "upcoming_activities", 1)
	assertRowCount(t, db, "vpp_app_upcoming_activities", 1)
	assertRowCount(t, db, "software_install_upcoming_activities", 0)
	assertRowCount(t, db, "script_upcoming_activities", 0)
	var execIDs []string
	err := db.Select(&execIDs, `SELECT execution_id FROM upcoming_activities`)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{execIDPending}, execIDs)
}

func TestUp_20250217093329_Load(t *testing.T) {
	t.Skip()
}
