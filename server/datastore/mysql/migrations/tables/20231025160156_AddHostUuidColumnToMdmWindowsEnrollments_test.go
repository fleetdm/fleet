package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20231025160156(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := `
      INSERT INTO mdm_windows_enrollments (
		mdm_device_id,
		mdm_hardware_id,
		device_state,
		device_type,
		device_name,
		enroll_type,
		enroll_user_id,
		enroll_proto_version,
		enroll_client_version,
		not_in_oobe ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	execNoErr(t, db, insertStmt, "devID", "hwID", "ds", "dt", "dn", "et", "euid", "epv", "ecv", 0)

	applyNext(t, db)

	// verify that the new column is present
	var mwed struct {
		ID                     uint      `db:"id"`
		MDMDeviceID            string    `db:"mdm_device_id"`
		MDMHardwareID          string    `db:"mdm_hardware_id"`
		MDMDeviceState         string    `db:"device_state"`
		MDMDeviceType          string    `db:"device_type"`
		MDMDeviceName          string    `db:"device_name"`
		MDMEnrollType          string    `db:"enroll_type"`
		MDMEnrollUserID        string    `db:"enroll_user_id"`
		MDMEnrollProtoVersion  string    `db:"enroll_proto_version"`
		MDMEnrollClientVersion string    `db:"enroll_client_version"`
		MDMNotInOOBE           bool      `db:"not_in_oobe"`
		CreatedAt              time.Time `db:"created_at"`
		UpdatedAt              time.Time `db:"updated_at"`
		HostUUID               string    `db:"host_uuid"`
	}
	err := db.Get(&mwed, "SELECT * FROM mdm_windows_enrollments WHERE mdm_device_id = ?", "devID")
	require.NoError(t, err)
	require.Equal(t, "devID", mwed.MDMDeviceID)
	require.Equal(t, "hwID", mwed.MDMHardwareID)
	require.Equal(t, "ds", mwed.MDMDeviceState)
	require.Equal(t, "dn", mwed.MDMDeviceName)
	require.Equal(t, "et", mwed.MDMEnrollType)
	require.Equal(t, "euid", mwed.MDMEnrollUserID)
	require.Equal(t, "epv", mwed.MDMEnrollProtoVersion)
	require.False(t, mwed.MDMNotInOOBE)
	require.Empty(t, mwed.HostUUID)

	insertStmt = `
      INSERT INTO mdm_windows_enrollments (
		mdm_device_id,
		mdm_hardware_id,
		device_state,
		device_type,
		device_name,
		enroll_type,
		enroll_user_id,
		enroll_proto_version,
		enroll_client_version,
		not_in_oobe,
		host_uuid ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	execNoErr(t, db, insertStmt, "devID2", "hwID2", "ds", "dt", "dn", "et", "euid", "epv", "ecv", 0, "hostUUID")

	err = db.Get(&mwed, "SELECT * FROM mdm_windows_enrollments WHERE mdm_device_id = ?", "devID2")
	require.NoError(t, err)
	require.Equal(t, "devID2", mwed.MDMDeviceID)
	require.Equal(t, "hwID2", mwed.MDMHardwareID)
	require.Equal(t, "ds", mwed.MDMDeviceState)
	require.Equal(t, "dn", mwed.MDMDeviceName)
	require.Equal(t, "et", mwed.MDMEnrollType)
	require.Equal(t, "euid", mwed.MDMEnrollUserID)
	require.Equal(t, "epv", mwed.MDMEnrollProtoVersion)
	require.False(t, mwed.MDMNotInOOBE)
	require.Equal(t, "hostUUID", mwed.HostUUID)
}
