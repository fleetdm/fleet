package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The migration only relaxes nullability and adds columns, so the risk worth testing is that it runs
// against populated tables and leaves existing rows intact with the documented defaults.
func TestUp_20260731213352(t *testing.T) {
	db := applyUpToPrev(t)

	const hostUUID = "existing-host"

	_, err := db.Exec(`
		INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, 'verified')`, hostUUID, []byte("enc"), "existing-command-uuid")
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO mdm_windows_enrollments (
			mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
			enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, host_uuid)
		VALUES ('device-1', 'hw-1', 'enrolled', 'CIMClient_Windows', 'DESKTOP-1', 'ProgrammaticEnrollment', '', '5.0', '10.0', ?)`,
		hostUUID)
	require.NoError(t, err)

	applyNext(t, db)

	var (
		password    []byte
		commandUUID string
		clientError string
	)
	require.NoError(t, db.QueryRow(
		`SELECT encrypted_password, command_uuid, client_error FROM host_managed_local_account_passwords WHERE host_uuid = ?`, hostUUID,
	).Scan(&password, &commandUUID, &clientError))
	require.Equal(t, []byte("enc"), password)
	require.Equal(t, "existing-command-uuid", commandUUID)
	require.Empty(t, clientError)

	var escrowed bool
	require.NoError(t, db.QueryRow(
		`SELECT managed_local_account_escrowed FROM mdm_windows_enrollments WHERE host_uuid = ?`, hostUUID,
	).Scan(&escrowed))
	require.False(t, escrowed)
}
