package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261027212007(t *testing.T) {
	db := applyUpToPrev(t)

	old := time.Now().UTC().Add(-30 * 24 * time.Hour).Truncate(time.Second)
	enrollEscrowed := func(name string, withHost bool) string {
		hostUUID := name + "-uuid"
		if withHost {
			execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'windows')`,
				name, name, name, hostUUID)
		}
		execNoErr(t, db, `
			INSERT INTO mdm_windows_enrollments (
				mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
				enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version,
				host_uuid, managed_local_account_escrowed, created_at, updated_at)
			VALUES (?, ?, 'enrolled', 'CIMClient_Windows', 'DESKTOP', 'ProgrammaticEnrollment', '', '5.0', '10.0', ?, 1, ?, ?)`,
			name, name, hostUUID, old, old)
		return hostUUID
	}
	insertPassword := func(hostUUID string, encryptedPassword any, deleted bool) {
		execNoErr(t, db, `INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, deleted) VALUES (?, ?, ?)`,
			hostUUID, encryptedPassword, deleted)
	}

	// The UUID changed after escrow, so nothing is stored under the current one.
	enrollEscrowed("drifted", true)
	insertPassword(enrollEscrowed("retired", true), "secret", true)
	// A failed attempt stores an error-only row with no password.
	insertPassword(enrollEscrowed("error-only", true), nil, false)
	insertPassword(enrollEscrowed("healthy", true), "secret", false)
	// Deleting a host leaves its enrollment behind.
	enrollEscrowed("deleted", false)

	applyNext(t, db)

	var stillEscrowed []string
	require.NoError(t, db.Select(&stillEscrowed,
		`SELECT mdm_device_id FROM mdm_windows_enrollments WHERE managed_local_account_escrowed = 1 ORDER BY mdm_device_id`))
	require.Equal(t, []string{"healthy"}, stillEscrowed)

	var touched int
	require.NoError(t, db.Get(&touched, `SELECT COUNT(*) FROM mdm_windows_enrollments WHERE updated_at != ?`, old))
	require.Zero(t, touched)
}
