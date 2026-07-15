package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260603120000(t *testing.T) {
	db := applyUpToPrev(t)

	insertEnrollment := func(deviceID string) int64 {
		res, err := db.Exec(`INSERT INTO mdm_windows_enrollments
			(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name, enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version)
			VALUES (?, ?, '', '', '', '', '', '', '')`, deviceID, deviceID+"-hw")
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	insertCommand := func(uuid string) {
		_, err := db.Exec(`INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, '<Get></Get>', './Some/Node')`, uuid)
		require.NoError(t, err)
	}
	queue := func(enrollID int64, uuid string) {
		_, err := db.Exec(`INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid) VALUES (?, ?)`, enrollID, uuid)
		require.NoError(t, err)
	}

	// A: an unacknowledged queued command -> should be backfilled to pending.
	enrollA := insertEnrollment("deviceA")
	insertCommand("cmd-a")
	queue(enrollA, "cmd-a")

	// B: a queued command that has already been acknowledged (a result exists) -> must NOT be marked pending.
	enrollB := insertEnrollment("deviceB")
	insertCommand("cmd-b")
	queue(enrollB, "cmd-b")
	respRes, err := db.Exec(`INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, '<resp/>')`, enrollB)
	require.NoError(t, err)
	respID, err := respRes.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, response_id, status_code)
		VALUES (?, 'cmd-b', '<r/>', ?, '200')`, enrollB, respID)
	require.NoError(t, err)

	// C: no queued commands -> stays not pending.
	enrollC := insertEnrollment("deviceC")

	// applyNext runs the ALTER plus the join-driven has_pending_commands backfill; this exercises that SQL on real data.
	applyNext(t, db)

	hasPending := func(enrollID int64) bool {
		var v bool
		require.NoError(t, db.Get(&v, `SELECT has_pending_commands FROM mdm_windows_enrollments WHERE id = ?`, enrollID))
		return v
	}
	require.True(t, hasPending(enrollA), "enrollment with an unacknowledged queued command should be backfilled to pending")
	require.False(t, hasPending(enrollB), "enrollment whose only queued command is acknowledged must not be pending")
	require.False(t, hasPending(enrollC), "enrollment with no queued commands must not be pending")
}
