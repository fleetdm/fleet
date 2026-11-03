package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261007155733(t *testing.T) {
	db := applyUpToPrev(t)

	const plist = `<?xml version="1.0"?><plist/>`
	const resultAt = "2026-02-02 02:02:02"
	const previouslyFailedAt = "2026-01-01 00:00:00"
	hostUUID := "host-uuid-1"
	hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES ('oh-1', 'nk-1', 'host1.local', ?, 'ios')`, hostUUID)
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, hostUUID)
	execNoErr(t, db, `
		INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
		VALUES (?, ?, 'Device', 'com.apple.mgmt.test', 'magic', 'abcdef', 1)`, hostUUID, hostUUID)
	appID := execNoErrLastID(t, db, `INSERT INTO in_house_apps (global_or_team_id, storage_id, platform, filename) VALUES (0, 'storage-1', 'ios', 'app.ipa')`)

	insertInstall := func(commandUUID string, resultStatus string, canceled bool, verificationFailed bool) {
		execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command, name) VALUES (?, 'InstallApplication', ?, '')`, commandUUID, plist)
		execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result, updated_at) VALUES (?, ?, ?, ?, ?)`, hostUUID, commandUUID, resultStatus, plist, resultAt)
		verificationFailedAt := "NULL"
		if verificationFailed {
			verificationFailedAt = "'" + previouslyFailedAt + "'"
		}
		execNoErr(t, db, `
			INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, platform, command_uuid, canceled, verification_failed_at)
			VALUES (?, ?, 'ios', ?, ?, `+verificationFailedAt+`)`, hostID, appID, commandUUID, canceled)
	}

	// insert a pending install whose command got Error
	insertInstall("errored", "Error", false, false)
	// insert a pending install whose command got CommandFormatError
	insertInstall("format-errored", "CommandFormatError", false, false)
	// insert a canceled install whose command got Error
	insertInstall("errored-canceled", "Error", true, false)
	// insert an install whose command got Error and that is already failed
	insertInstall("errored-already-failed", "Error", false, true)
	// insert a pending install whose command was acknowledged
	insertInstall("acknowledged", "Acknowledged", false, false)
	// move the next id past the first batch of 1000 and insert a pending install whose command got Error
	execNoErr(t, db, `ALTER TABLE host_in_house_software_installs AUTO_INCREMENT = 1500`)
	insertInstall("errored-second-batch", "Error", false, false)

	applyNext(t, db)

	const selectFailedAt = `SELECT CAST(verification_failed_at AS CHAR(19)) FROM host_in_house_software_installs WHERE command_uuid = ?`
	var failedAt *string

	// the install whose command got Error should be failed at the time of the result
	require.NoError(t, db.Get(&failedAt, selectFailedAt, "errored"))
	require.NotNil(t, failedAt)
	require.Equal(t, resultAt, *failedAt)

	// the install whose command got CommandFormatError should be failed at the time of the result
	require.NoError(t, db.Get(&failedAt, selectFailedAt, "format-errored"))
	require.NotNil(t, failedAt)
	require.Equal(t, resultAt, *failedAt)

	// the canceled install should not be failed
	require.NoError(t, db.Get(&failedAt, selectFailedAt, "errored-canceled"))
	require.Nil(t, failedAt)

	// the already failed install should keep its failed time
	require.NoError(t, db.Get(&failedAt, selectFailedAt, "errored-already-failed"))
	require.NotNil(t, failedAt)
	require.Equal(t, previouslyFailedAt, *failedAt)

	// the acknowledged install should not be failed
	require.NoError(t, db.Get(&failedAt, selectFailedAt, "acknowledged"))
	require.Nil(t, failedAt)

	// the install in the second batch should be failed at the time of the result
	require.NoError(t, db.Get(&failedAt, selectFailedAt, "errored-second-batch"))
	require.NotNil(t, failedAt)
	require.Equal(t, resultAt, *failedAt)
}
