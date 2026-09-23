package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260923140854(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	newEnrollment := func(hardwareID string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO mdm_windows_enrollments
			(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
			 enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version)
			VALUES (?, ?, 'state', 'CIMClient_Windows', 'DESKTOP-1', 'Device', 'upn@example.com', '4.0', '10.0')`,
			"device-"+hardwareID, hardwareID)
	}

	enrollmentID := newEnrollment("hw-1")

	// A Windows secret is minted before any hosts row exists, so it binds to the enrollment and leaves host_id NULL.
	execNoErr(t, db, `INSERT INTO host_one_time_enroll_secrets
		(secret, host_id, mdm_windows_enrollment_id, platform, hardware_uuid, hardware_serial)
		VALUES ('win-secret', NULL, ?, 'windows', '', '')`, enrollmentID)

	var bound int
	require.NoError(t, db.Get(&bound, `SELECT COUNT(*) FROM host_one_time_enroll_secrets
		WHERE mdm_windows_enrollment_id = ? AND host_id IS NULL`, enrollmentID))
	require.Equal(t, 1, bound)

	// The Apple path keeps working: no enrollment binding, host_id set.
	execNoErr(t, db, `INSERT INTO host_one_time_enroll_secrets
		(secret, host_id, platform, hardware_uuid, hardware_serial)
		VALUES ('mac-secret', 1, 'darwin', 'UUID-1', 'SERIAL-1')`)

	// The binding must reference a real enrollment.
	_, err := db.Exec(`INSERT INTO host_one_time_enroll_secrets
		(secret, mdm_windows_enrollment_id, platform, hardware_uuid, hardware_serial)
		VALUES ('orphan', 999999, 'windows', '', '')`)
	require.Error(t, err, "a secret must not bind to a nonexistent enrollment")

	// Re-enrollment deletes the enrollment row, and that is what invalidates the secret minted for it. The Apple row,
	// which has no binding, must survive.
	execNoErr(t, db, `DELETE FROM mdm_windows_enrollments WHERE id = ?`, enrollmentID)

	var remaining []string
	require.NoError(t, db.Select(&remaining, `SELECT secret FROM host_one_time_enroll_secrets ORDER BY secret`))
	require.Equal(t, []string{"mac-secret"}, remaining, "the Windows secret must cascade away with its enrollment")
}
