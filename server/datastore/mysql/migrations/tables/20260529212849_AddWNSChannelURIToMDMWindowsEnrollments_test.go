package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260529212849(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert an enrollment before the migration adds the new columns.
	_, err := db.Exec(`
		INSERT INTO mdm_windows_enrollments
			(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
			 enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version)
		VALUES ('dev-1', 'hw-1', 'managed', 'CIMClient_Windows', 'DESKTOP-1', 'Device', 'user-1', '4.0', '10.0.0')`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// The existing row gets NULL for all three new columns.
	var (
		channelURI sql.NullString
		status     sql.NullInt64
		updatedAt  sql.NullTime
	)
	err = db.QueryRow(`SELECT wns_channel_uri, wns_channel_uri_status, wns_channel_uri_updated_at
		FROM mdm_windows_enrollments WHERE mdm_device_id = 'dev-1'`).Scan(&channelURI, &status, &updatedAt)
	require.NoError(t, err)
	assert.False(t, channelURI.Valid)
	assert.False(t, status.Valid)
	assert.False(t, updatedAt.Valid)

	// The new columns are writable and read back correctly.
	const uri = "https://db5.notify.windows.com/?token=abc123"
	_, err = db.Exec(`UPDATE mdm_windows_enrollments
		SET wns_channel_uri = ?, wns_channel_uri_status = 0, wns_channel_uri_updated_at = NOW(6)
		WHERE mdm_device_id = 'dev-1'`, uri)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT wns_channel_uri, wns_channel_uri_status, wns_channel_uri_updated_at
		FROM mdm_windows_enrollments WHERE mdm_device_id = 'dev-1'`).Scan(&channelURI, &status, &updatedAt)
	require.NoError(t, err)
	assert.Equal(t, uri, channelURI.String)
	assert.True(t, status.Valid)
	assert.EqualValues(t, 0, status.Int64)
	assert.True(t, updatedAt.Valid)
}
