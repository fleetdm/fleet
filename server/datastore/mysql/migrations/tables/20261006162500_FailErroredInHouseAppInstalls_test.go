package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261006162500(t *testing.T) {
	db := applyUpToPrev(t)

	const plist = `<?xml version="1.0"?><plist/>`
	hostUUID := "host-uuid-1"
	hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES ('oh-1', 'nk-1', 'host1.local', ?, 'ios')`, hostUUID)
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, hostUUID)
	execNoErr(t, db, `
		INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
		VALUES (?, ?, 'Device', 'com.apple.mgmt.test', 'magic', 'abcdef', 1)`, hostUUID, hostUUID)
	appID := execNoErrLastID(t, db, `INSERT INTO in_house_apps (global_or_team_id, storage_id, platform, filename) VALUES (0, 'storage-1', 'ios', 'app.ipa')`)

	insertInstall := func(commandUUID string, resultStatus string, canceled bool, verificationFailed bool) {
		execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command, name) VALUES (?, 'InstallApplication', ?, '')`, commandUUID, plist)
		execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, ?)`, hostUUID, commandUUID, resultStatus, plist)
		verificationFailedAt := "NULL"
		if verificationFailed {
			verificationFailedAt = "'2026-01-01 00:00:00'"
		}
		execNoErr(t, db, `
			INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, platform, command_uuid, canceled, verification_failed_at)
			VALUES (?, ?, 'ios', ?, ?, `+verificationFailedAt+`)`, hostID, appID, commandUUID, canceled)
	}

	insertInstall("errored", "Error", false, false)
	insertInstall("errored-canceled", "Error", true, false)
	insertInstall("errored-already-failed", "Error", false, true)
	insertInstall("acknowledged", "Acknowledged", false, false)
	insertInstall("format-errored", "CommandFormatError", false, false)
	// move the next id past the first batch of 1000
	execNoErr(t, db, `ALTER TABLE host_in_house_software_installs AUTO_INCREMENT = 1500`)
	insertInstall("errored-second-batch", "Error", false, false)

	applyNext(t, db)

	var failedAtByCommand []struct {
		CommandUUID string  `db:"command_uuid"`
		FailedAt    *string `db:"failed_at"`
	}
	require.NoError(t, db.Select(&failedAtByCommand, `SELECT command_uuid, CAST(verification_failed_at AS CHAR) AS failed_at FROM host_in_house_software_installs ORDER BY id`))
	require.Len(t, failedAtByCommand, 6)

	require.Equal(t, "errored", failedAtByCommand[0].CommandUUID)
	require.NotNil(t, failedAtByCommand[0].FailedAt)
	require.Equal(t, "errored-canceled", failedAtByCommand[1].CommandUUID)
	require.Nil(t, failedAtByCommand[1].FailedAt)
	require.Equal(t, "errored-already-failed", failedAtByCommand[2].CommandUUID)
	require.NotNil(t, failedAtByCommand[2].FailedAt)
	require.Contains(t, *failedAtByCommand[2].FailedAt, "2026-01-01 00:00:00")
	require.Equal(t, "acknowledged", failedAtByCommand[3].CommandUUID)
	require.Nil(t, failedAtByCommand[3].FailedAt)
	require.Equal(t, "format-errored", failedAtByCommand[4].CommandUUID)
	require.NotNil(t, failedAtByCommand[4].FailedAt)
	require.Equal(t, "errored-second-batch", failedAtByCommand[5].CommandUUID)
	require.NotNil(t, failedAtByCommand[5].FailedAt)
}
