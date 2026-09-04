package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260902143818(t *testing.T) {
	db := applyUpToPrev(t)

	// An enrollment that predates the column, to prove the default applies to existing rows.
	execNoErrLastID(t, db, `
		INSERT INTO mdm_windows_enrollments (
			mdm_device_id, mdm_hardware_id, device_state, device_type, device_name, enroll_type,
			enroll_user_id, enroll_proto_version, enroll_client_version, not_in_oobe, host_uuid
		) VALUES ('d1', 'hw1', 'enrolled', 'CIMClient_Windows', 'DESKTOP-1', 'ProgrammaticEnrollment',
			'', '5.0', '10.0.0.0', 0, 'HOST-UUID-1')`)

	applyNext(t, db)

	var requested bool
	require.NoError(t, db.Get(&requested,
		`SELECT managed_local_account_rotation_requested FROM mdm_windows_enrollments WHERE host_uuid = 'HOST-UUID-1'`))
	require.False(t, requested)

	// And that the column is writable.
	execNoErr(t, db,
		`UPDATE mdm_windows_enrollments SET managed_local_account_rotation_requested = 1 WHERE host_uuid = 'HOST-UUID-1'`)
	require.NoError(t, db.Get(&requested,
		`SELECT managed_local_account_rotation_requested FROM mdm_windows_enrollments WHERE host_uuid = 'HOST-UUID-1'`))
	require.True(t, requested)
}
