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
	contentIDs := insertScriptContents(t, db, 1)
	scriptContentID := contentIDs[0]

	// insert a couple pending but one has host_deleted_at set, and a non-pending script
	execIDPending, execIDDeleted, execIDDone := uuid.NewString(), uuid.NewString(), uuid.NewString()
	execNoErr(t, db, `INSERT INTO host_script_results
		(host_id, execution_id, output, script_content_id, host_deleted_at)
	VALUES (?, ?, '', ?, ?)`, hostID, execIDPending, scriptContentID, nil)
	execNoErr(t, db, `INSERT INTO host_script_results
		(host_id, execution_id, output, script_content_id, host_deleted_at)
	VALUES (?, ?, '', ?, ?)`, hostID, execIDDeleted, scriptContentID, time.Now())
	execNoErr(t, db, `INSERT INTO host_script_results
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

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM upcoming_activities WHERE activated_at IS NULL`)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
}

func TestUp_20250217093329_SoftwareInstall(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)

	installerIDs, _ := insertSoftwareInstallers(t, db, 1)
	installerID := installerIDs[0]

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

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM upcoming_activities WHERE activated_at IS NULL`)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
}

func TestUp_20250217093329_SoftwareUninstall(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)

	installerIDs, _ := insertSoftwareInstallers(t, db, 1)
	installerID := installerIDs[0]

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

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM upcoming_activities WHERE activated_at IS NULL`)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
}

func TestUp_20250217093329_VPPInstall(t *testing.T) {
	db := applyUpToPrev(t)
	hostID := insertHost(t, db, nil)
	hostUUID := "12345678-1234-1234-1234-123456789012"

	adamIDs, _ := insertVPPApps(t, db, 1, "darwin")
	adamID := adamIDs[0]

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

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM upcoming_activities WHERE activated_at IS NULL`)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
}

func TestUp_20250217093329_Load(t *testing.T) {
	db := applyUpToPrev(t)

	// create a 1000 hosts each for macOS, Windows and Linux
	macIDs, winIDs, linuxIDs, idsToUUIDs := insertHosts(t, db, 1000, 1000, 1000)

	// create 10 scripts
	scriptContentIDs := insertScriptContents(t, db, 10)
	// create 10 software installers/uninstallers
	installerIDs, _ := insertSoftwareInstallers(t, db, 10)
	// create 10 VPP apps
	adamIDs, _ := insertVPPApps(t, db, 10, "darwin")

	// for each host, create a pending script execution, software install, software
	// uninstall, and for macOS hosts create a VPP app install.
	var allExecIDs []string
	perPlatformIDs := map[string][]uint{"darwin": macIDs, "windows": winIDs, "linux": linuxIDs}
	for platform, hostIDs := range perPlatformIDs {
		for i, hostID := range hostIDs {
			// create the pending script
			execID := uuid.NewString()
			execNoErr(t, db, `INSERT INTO host_script_results
				(host_id, execution_id, output, script_content_id)
				VALUES (?, ?, '', ?)`, hostID, execID, scriptContentIDs[i%len(scriptContentIDs)])
			allExecIDs = append(allExecIDs, execID)

			// create the pending software install
			execID = uuid.NewString()
			execNoErr(t, db, `INSERT INTO host_software_installs
				(host_id, execution_id, software_installer_id) VALUES (?, ?, ?)`,
				hostID, execID, installerIDs[i%len(installerIDs)])
			allExecIDs = append(allExecIDs, execID)

			// create the pending software uninstall
			execID = uuid.NewString()
			execNoErr(t, db, `INSERT INTO host_software_installs
				(host_id, execution_id, software_installer_id, uninstall) VALUES (?, ?, ?, 1)`,
				hostID, execID, installerIDs[(i+1)%len(installerIDs)])
			allExecIDs = append(allExecIDs, execID)

			if platform == "darwin" {
				execID = uuid.NewString()
				execNoErr(t, db, `INSERT INTO host_vpp_software_installs
					(host_id, adam_id, platform, command_uuid) VALUES (?, ?, ?, ?)`,
					hostID, adamIDs[i%len(adamIDs)], "darwin", execID)
				execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`,
					execID, "InstallApplication", "<?xml")
				execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`,
					idsToUUIDs[hostID], execID)
				allExecIDs = append(allExecIDs, execID)
			}
		}
	}

	applyNext(t, db)
	assertRowCount(t, db, "host_vpp_software_installs", 1000)
	assertRowCount(t, db, "host_script_results", 3000)
	assertRowCount(t, db, "host_software_installs", 6000)
	assertRowCount(t, db, "upcoming_activities", 10000)
	assertRowCount(t, db, "vpp_app_upcoming_activities", 1000)
	assertRowCount(t, db, "software_install_upcoming_activities", 6000)
	assertRowCount(t, db, "script_upcoming_activities", 3000)

	var execIDs []string
	err := db.Select(&execIDs, `SELECT execution_id FROM upcoming_activities`)
	require.NoError(t, err)
	require.Len(t, execIDs, len(allExecIDs))
	require.ElementsMatch(t, allExecIDs, execIDs)

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM upcoming_activities WHERE activated_at IS NULL`)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)

	var stats []struct {
		TargetID    int    `db:"target_id"`
		TargetIDStr string `db:"target_id_str"`
		Count       int    `db:"count"`
	}
	// sanity-check software installs
	err = db.Select(&stats, `SELECT software_installer_id as target_id, COUNT(DISTINCT host_id) as count
		FROM upcoming_activities ua INNER JOIN software_install_upcoming_activities siua
		ON ua.id = siua.upcoming_activity_id
		WHERE ua.activity_type = 'software_install'
		GROUP BY software_installer_id`)
	require.NoError(t, err)
	require.Len(t, stats, 10)
	for _, stat := range stats {
		// each installer installs on 1/10th of the hosts
		require.EqualValues(t, 300, stat.Count)
	}

	// sanity-check software uninstalls
	stats = stats[:0]
	err = db.Select(&stats, `SELECT software_installer_id as target_id, COUNT(DISTINCT host_id) as count
		FROM upcoming_activities ua INNER JOIN software_install_upcoming_activities siua
		ON ua.id = siua.upcoming_activity_id
		WHERE ua.activity_type = 'software_uninstall'
		GROUP BY software_installer_id`)
	require.NoError(t, err)
	require.Len(t, stats, 10)
	for _, stat := range stats {
		// each installer uninstalls on 1/10th of the hosts
		require.EqualValues(t, 300, stat.Count)
	}

	// sanity-check scripts
	stats = stats[:0]
	err = db.Select(&stats, `SELECT script_content_id as target_id, COUNT(DISTINCT host_id) as count
		FROM upcoming_activities ua INNER JOIN script_upcoming_activities sua
		ON ua.id = sua.upcoming_activity_id
		WHERE ua.activity_type = 'script'
		GROUP BY script_content_id`)
	require.NoError(t, err)
	require.Len(t, stats, 10)
	for _, stat := range stats {
		// each script runs on 1/10th of the hosts
		require.EqualValues(t, 300, stat.Count)
	}

	// sanity-check VPP apps
	stats = stats[:0]
	err = db.Select(&stats, `SELECT adam_id as target_id_str, COUNT(DISTINCT host_id) as count
		FROM upcoming_activities ua INNER JOIN vpp_app_upcoming_activities vaua
		ON ua.id = vaua.upcoming_activity_id
		WHERE ua.activity_type = 'vpp_app_install'
		GROUP BY adam_id`)
	require.NoError(t, err)
	require.Len(t, stats, 10)
	for _, stat := range stats {
		// each vpp app installs on 1/10th of the macOS hosts
		require.EqualValues(t, 100, stat.Count)
	}
}
