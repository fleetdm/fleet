package tables

import (
	"crypto/md5" //nolint:gosec // we are using MD5 here to match the hash sent by Windows devices
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20260126150840(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Grab an MD5 hashed random password phrase
	deviceId := "device123"
	password := "dummy"
	dummy := md5.Sum([]byte(deviceId + ":" + password)) //nolint:gosec // we are using MD5 here to match the hash sent by Windows devices
	_, err := db.Exec(`INSERT INTO mdm_windows_enrollments (mdm_device_id, mdm_hardware_id, device_state, device_type, device_name, enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, not_in_oobe, credentials_hash, credentials_acknowledged, host_uuid) VALUES (?, "bogus", "MDMDeviceEnrolledEnrolled", "CIMClient_Windows", "name", "Full", "bogus", "7.0", "10", 0, ?, ?, ?)`, deviceId, dummy[:], true, uuid.NewString())
	require.NoError(t, err)

	var rows []struct {
		DeviceID     string `db:"mdm_device_id"`
		Hash         []byte `db:"credentials_hash"`
		Acknowledged bool   `db:"credentials_acknowledged"`
	}
	err = db.Select(&rows, "SELECT mdm_device_id, credentials_hash, credentials_acknowledged FROM mdm_windows_enrollments WHERE mdm_device_id = ?", deviceId)
	require.NoError(t, err)

	require.Len(t, rows, 1)
	require.NotEqual(t, rows[0].Hash, "dummy")

	// Rehash the same (emulate coming from the app)
	creds := md5.Sum([]byte(deviceId + ":" + password)) //nolint:gosec // we are using MD5 here to match the hash sent by Windows devices

	// We will recieve a b64 encoded MD5 hash from the app, so we do string comparison here, as that is most likely how it will be
	require.True(t, string(rows[0].Hash) == string(creds[:]))

	// We have to use a slice here to convert to the same type (no size []byte)
	require.Equal(t, creds[:], rows[0].Hash)
}
