package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260606051849(t *testing.T) {
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

	// A: queued, never acknowledged -> acked_at must stay NULL (still pending).
	enrollA := insertEnrollment("deviceA")
	insertCommand("cmd-a")
	queue(enrollA, "cmd-a")

	// B: queued and acknowledged before the migration (result row exists) -> acked_at must be backfilled from the
	// result's created_at so the row does not reappear as pending.
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

	applyNext(t, db)

	ackedAt := func(enrollID int64, uuid string) *string {
		var v *string
		require.NoError(t, db.Get(&v, `SELECT acked_at FROM windows_mdm_command_queue WHERE enrollment_id = ? AND command_uuid = ?`, enrollID, uuid))
		return v
	}
	require.Nil(t, ackedAt(enrollA, "cmd-a"), "unacknowledged queue row must keep acked_at NULL")
	ackedB := ackedAt(enrollB, "cmd-b")
	require.NotNil(t, ackedB, "acknowledged queue row must be backfilled with acked_at")

	// Compare in SQL rather than as strings: created_at is a second-resolution timestamp while acked_at is DATETIME(6),
	// so the driver renders them with different fractional-second suffixes.
	var backfillMatches bool
	require.NoError(t, db.Get(&backfillMatches, `SELECT q.acked_at = r.created_at
		FROM windows_mdm_command_queue q
		JOIN windows_mdm_command_results r ON r.enrollment_id = q.enrollment_id AND r.command_uuid = q.command_uuid
		WHERE q.enrollment_id = ? AND q.command_uuid = 'cmd-b'`, enrollB))
	require.True(t, backfillMatches, "backfilled acked_at must equal the result row's created_at")
}
