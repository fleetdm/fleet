package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260831204156(t *testing.T) {
	db := applyUpToPrev(t)

	const sharedHardwareID = "F19B99942A3B9C53679F46017599C1FC4953B9BD3F0BC20FF681D0F93FAE5992"

	insertEnrollment := func(deviceID, hardwareID, hostUUID string) error {
		_, err := db.Exec(`
			INSERT INTO mdm_windows_enrollments (
				mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
				enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, host_uuid
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			deviceID, hardwareID, "MDMDeviceEnrolledEnrolled", "CIMClient_Windows", "REPRO",
			"ProgrammaticEnrollment", "", "5.0", "10.0.26200.8037", hostUUID)
		return err
	}

	// A pre-migration row must survive the index swap untouched.
	require.NoError(t, insertEnrollment("device-a", sharedHardwareID, "host-uuid-a"))

	// Before the migration, a second host presenting the same hardware ID is rejected outright. That rejection is what
	// forced the enrollment path to delete the incumbent row, which is the silent unenrollment this migration undoes.
	err := insertEnrollment("device-b", sharedHardwareID, "host-uuid-b")
	require.Error(t, err, "the old single-column unique key should reject a second host on the same hardware ID")

	applyNext(t, db)

	var count int
	require.NoError(t, db.Get(&count,
		`SELECT COUNT(*) FROM mdm_windows_enrollments WHERE mdm_hardware_id = ?`, sharedHardwareID))
	require.Equal(t, 1, count, "the pre-migration row must survive the migration")

	// The point of the migration: two different hosts may now hold their own enrollment under one hardware ID.
	require.NoError(t, insertEnrollment("device-b", sharedHardwareID, "host-uuid-b"),
		"a second host on the same hardware ID should be accepted after the migration")

	// One host still cannot hold two rows for the same hardware ID, which is what keeps the per-host enrollment unique
	// and preserves the "newest row per host_uuid" resolution the command queue relies on.
	err = insertEnrollment("device-c", sharedHardwareID, "host-uuid-a")
	require.Error(t, err, "the same host re-presenting the same hardware ID must still be rejected")

	require.NoError(t, db.Get(&count,
		`SELECT COUNT(DISTINCT host_uuid) FROM mdm_windows_enrollments WHERE mdm_hardware_id = ?`, sharedHardwareID))
	require.Equal(t, 2, count, "both hosts should be enrolled under the shared hardware ID")
}
