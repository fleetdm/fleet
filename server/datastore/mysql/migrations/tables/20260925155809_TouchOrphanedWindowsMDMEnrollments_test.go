package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260925155809(t *testing.T) {
	db := applyUpToPrev(t)

	// Force several batches so the id cursor has to cross batch boundaries.
	prev := touchOrphanedWindowsMDMEnrollmentsBatchSize
	touchOrphanedWindowsMDMEnrollmentsBatchSize = 2
	t.Cleanup(func() { touchOrphanedWindowsMDMEnrollmentsBatchSize = prev })

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
	// Inserted in id order, so with a batch size of 2 the orphans land in
	// every batch and the kept rows sit between them.
	insert("orphaned-1", "deleted-uuid-1")
	insert("linked", "live-uuid")
	insert("orphaned-2", "deleted-uuid-2")
	insert("unlinked", "")
	insert("orphaned-3", "deleted-uuid-3")

	applyNext(t, db)

	updatedAt := func(deviceID string) time.Time {
		var ts time.Time
		require.NoError(t, db.Get(&ts, `SELECT updated_at FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, deviceID))
		return ts.UTC()
	}
	// A wide margin absorbs clock drift between Go and the MySQL server.
	for _, id := range []string{"orphaned-1", "orphaned-2", "orphaned-3"} {
		require.WithinDuration(t, time.Now().UTC(), updatedAt(id), time.Hour, "%s gets a fresh window", id)
	}
	require.Equal(t, old, updatedAt("linked"), "linked enrollment untouched")
	require.Equal(t, old, updatedAt("unlinked"), "never-linked enrollment untouched")
}
