package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261027212007(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(name, hostUUID string) {
		execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'windows')`,
			"oh-"+name, "nk-"+name, name, hostUUID)
	}
	old := time.Now().UTC().Add(-30 * 24 * time.Hour).Truncate(time.Second)
	insertEnrollment := func(deviceID, hostUUID string, escrowed bool) {
		execNoErr(t, db, `
			INSERT INTO mdm_windows_enrollments (
				mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
				enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version,
				host_uuid, managed_local_account_escrowed, created_at, updated_at)
			VALUES (?, ?, 'enrolled', 'CIMClient_Windows', 'DESKTOP', 'ProgrammaticEnrollment', '', '5.0', '10.0', ?, ?, ?, ?)`,
			deviceID, "hw-"+deviceID, hostUUID, escrowed, old, old)
	}
	insertPassword := func(hostUUID string, deleted bool) {
		execNoErr(t, db, `INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, status, deleted)
			VALUES (?, 'secret', 'verified', ?)`, hostUUID, deleted)
	}
	insertEscrowError := func(hostUUID string) {
		execNoErr(t, db, `INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, status, client_error)
			VALUES (?, NULL, 'failed', 'access denied')`, hostUUID)
	}

	// The host's UUID changed after escrow, so its live password is under the old UUID.
	insertHost("drifted", "drifted-new-uuid")
	insertEnrollment("drifted", "drifted-new-uuid", true)
	insertPassword("drifted-old-uuid", false)

	// Only a retired password under the current UUID, e.g. from an earlier enrollment.
	insertHost("retired", "retired-uuid")
	insertEnrollment("retired", "retired-uuid", true)
	insertPassword("retired-uuid", true)

	// The UUID drifted and a later failed attempt left an error-only row, with no password, under the current UUID.
	insertHost("error-only", "error-only-new-uuid")
	insertEnrollment("error-only", "error-only-new-uuid", true)
	insertPassword("error-only-old-uuid", false)
	insertEscrowError("error-only-new-uuid")

	insertHost("healthy", "healthy-uuid")
	insertEnrollment("healthy", "healthy-uuid", true)
	insertPassword("healthy-uuid", false)

	// The host was deleted, which leaves its enrollment and password behind.
	insertEnrollment("deleted", "deleted-uuid", true)

	insertHost("never-escrowed", "never-escrowed-uuid")
	insertEnrollment("never-escrowed", "never-escrowed-uuid", false)

	applyNext(t, db)

	want := map[string]bool{
		"drifted":        false,
		"retired":        false,
		"error-only":     false,
		"healthy":        true,
		"deleted":        true,
		"never-escrowed": false,
	}
	for deviceID, wantEscrowed := range want {
		t.Run(deviceID, func(t *testing.T) {
			var row struct {
				Escrowed  bool      `db:"managed_local_account_escrowed"`
				UpdatedAt time.Time `db:"updated_at"`
			}
			require.NoError(t, db.Get(&row, `SELECT managed_local_account_escrowed, updated_at FROM mdm_windows_enrollments WHERE mdm_device_id = ?`,
				deviceID))
			require.Equal(t, wantEscrowed, row.Escrowed)
			require.True(t, old.Equal(row.UpdatedAt), "updated_at changed: %s", row.UpdatedAt)
		})
	}

	// Passwords are left alone, including the stranded one.
	var livePasswords []string
	require.NoError(t, db.Select(&livePasswords, `SELECT host_uuid FROM host_managed_local_account_passwords WHERE deleted = 0 ORDER BY host_uuid`))
	require.Equal(t, []string{"drifted-old-uuid", "error-only-new-uuid", "error-only-old-uuid", "healthy-uuid"}, livePasswords)
}
