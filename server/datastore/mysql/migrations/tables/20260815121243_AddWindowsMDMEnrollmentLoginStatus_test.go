package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260815121243(t *testing.T) {
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

	// An enrollment that predates the migration must come out with a NULL observation, which the gate reads as "never
	// observed" and therefore holds on rather than treating as "no user signed in".
	existing := insertEnrollment("device-existing")

	applyNext(t, db)

	var status *string
	var statusAt *string
	require.NoError(t, db.QueryRow(
		`SELECT last_login_status, last_login_status_at FROM mdm_windows_enrollments WHERE id = ?`, existing,
	).Scan(&status, &statusAt))
	require.Nil(t, status, "pre-existing enrollment must have no login status observation")
	require.Nil(t, statusAt, "pre-existing enrollment must have no login status timestamp")

	// New rows still insert without naming the columns, and the columns accept the wire values.
	fresh := insertEnrollment("device-fresh")
	_, err := db.Exec(`UPDATE mdm_windows_enrollments SET last_login_status = ?, last_login_status_at = NOW(6) WHERE id = ?`, "others", fresh)
	require.NoError(t, err)

	require.NoError(t, db.QueryRow(
		`SELECT last_login_status FROM mdm_windows_enrollments WHERE id = ?`, fresh,
	).Scan(&status))
	require.NotNil(t, status)
	require.Equal(t, "others", *status)
}
