package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260925155809(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform)
		VALUES ('oh-live', 'nk-live', 'win-live', 'live-uuid', 'windows')`)

	old := time.Now().UTC().Add(-90 * 24 * time.Hour).Truncate(time.Second)
	insert := func(deviceID, hostUUID string) {
		execNoErr(t, db, `
			INSERT INTO mdm_windows_enrollments (
				mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
				enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version,
				host_uuid, created_at, updated_at)
			VALUES (?, ?, 'enrolled', 'CIMClient_Windows', 'DESKTOP', 'ProgrammaticEnrollment', '', '5.0', '10.0', ?, ?, ?)`,
			deviceID, "hw-"+deviceID, hostUUID, old, old)
	}
	insert("orphaned", "deleted-uuid")
	insert("linked", "live-uuid")
	insert("unlinked", "")

	applyNext(t, db)

	updatedAt := func(deviceID string) time.Time {
		var ts time.Time
		require.NoError(t, db.Get(&ts, `SELECT updated_at FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, deviceID))
		return ts.UTC()
	}
	require.WithinDuration(t, time.Now().UTC(), updatedAt("orphaned"), time.Hour, "orphaned enrollment gets a fresh window")
	require.Equal(t, old, updatedAt("linked"), "linked enrollment untouched")
	require.Equal(t, old, updatedAt("unlinked"), "never-linked enrollment untouched")
}
