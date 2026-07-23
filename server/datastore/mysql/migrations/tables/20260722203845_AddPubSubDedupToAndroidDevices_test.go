package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260722203845(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a host and its android_devices row before the migration.
	res, err := db.Exec(`INSERT INTO hosts (hostname, uuid, platform, team_id, osquery_host_id, node_key,
		detail_updated_at, label_updated_at, policy_updated_at)
		VALUES ('android1', 'uuid1', 'android', NULL, 'oq1', 'nk1', '2026-01-01', '2026-01-01', '2026-01-01')`)
	require.NoError(t, err)
	hostID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, 'd1', 'esid1')`, hostID)
	require.NoError(t, err)

	// Apply migration.
	applyNext(t, db)

	// Existing rows have NULL for both new columns.
	var messageID *string
	require.NoError(t, db.Get(&messageID, `SELECT last_pubsub_message_id FROM android_devices WHERE device_id = 'd1'`))
	require.Nil(t, messageID)

	var eventTime *string
	require.NoError(t, db.Get(&eventTime, `SELECT last_pubsub_event_time FROM android_devices WHERE device_id = 'd1'`))
	require.Nil(t, eventTime)

	// The columns are writable and round-trip.
	_, err = db.Exec(`UPDATE android_devices
		SET last_pubsub_message_id = 'msg-123', last_pubsub_event_time = '2026-07-22 10:00:00.000000'
		WHERE device_id = 'd1'`)
	require.NoError(t, err)

	require.NoError(t, db.Get(&messageID, `SELECT last_pubsub_message_id FROM android_devices WHERE device_id = 'd1'`))
	require.NotNil(t, messageID)
	require.Equal(t, "msg-123", *messageID)

	require.NoError(t, db.Get(&eventTime, `SELECT last_pubsub_event_time FROM android_devices WHERE device_id = 'd1'`))
	require.NotNil(t, eventTime)
}
