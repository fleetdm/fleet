package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261026111120(t *testing.T) {
	db := applyUpToPrev(t)

	const plist = `<?xml version="1.0"?><plist/>`
	enroll := func(id string) {
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, id)
		execNoErr(t, db, `
			INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
			VALUES (?, ?, 'Device', 'com.apple.mgmt.test', 'magic', 'abcdef', 1)`, id, id)
	}
	// queues a command for the host, created at the given time, with an optional result status
	queue := func(id, cmdUUID, requestType, createdAt, resultStatus string) {
		execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`, cmdUUID, requestType, plist)
		execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid, created_at) VALUES (?, ?, ?)`, id, cmdUUID, createdAt)
		if resultStatus != "" {
			execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, ?)`,
				id, cmdUUID, resultStatus, plist)
		}
	}
	// a row as 4.92 left it: pending, with none of the command UUIDs
	legacyRow := func(id, operationType string, current, pending []byte) {
		execNoErr(t, db, `
			INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, pending_encrypted_password, status, operation_type)
			VALUES (?, ?, ?, 'pending', ?)`, id, current, pending, operationType)
	}
	old, rotated := []byte("old"), []byte("new")

	for _, id := range []string{
		"rotation-unanswered", "first-notnow", "rotation-acked", "first-acked", "clear-acked",
		"first-error", "rotation-error", "no-command", "two-sets", "tracked", "verified", "deleted",
	} {
		enroll(id)
	}

	legacyRow("rotation-unanswered", "install", old, rotated)
	queue("rotation-unanswered", "set-1", "SetRecoveryLock", "2026-09-01 10:00:00", "")

	legacyRow("first-notnow", "install", rotated, nil)
	queue("first-notnow", "set-2", "SetRecoveryLock", "2026-09-01 10:00:00", "NotNow")

	legacyRow("rotation-acked", "install", old, rotated)
	queue("rotation-acked", "set-3", "SetRecoveryLock", "2026-09-01 10:00:00", "Acknowledged")

	legacyRow("first-acked", "install", rotated, nil)
	queue("first-acked", "set-4", "SetRecoveryLock", "2026-09-01 10:00:00", "Acknowledged")

	legacyRow("clear-acked", "remove", old, nil)
	queue("clear-acked", "set-5", "SetRecoveryLock", "2026-09-01 10:00:00", "Acknowledged")

	legacyRow("first-error", "install", rotated, nil)
	queue("first-error", "set-6", "SetRecoveryLock", "2026-09-01 10:00:00", "Error")

	legacyRow("rotation-error", "install", old, rotated)
	queue("rotation-error", "set-7", "SetRecoveryLock", "2026-09-01 10:00:00", "Error")

	legacyRow("no-command", "install", old, rotated)
	queue("no-command", "profile-1", "InstallProfile", "2026-09-01 11:00:00", "")

	// only the latest SetRecoveryLock counts
	legacyRow("two-sets", "install", old, rotated)
	queue("two-sets", "set-8", "SetRecoveryLock", "2026-09-01 09:00:00", "Acknowledged")
	queue("two-sets", "set-9", "SetRecoveryLock", "2026-09-01 10:00:00", "")

	// rows written after the command tracking existed, and a deleted legacy row, are left alone
	queue("tracked", "set-10", "SetRecoveryLock", "2026-09-01 10:00:00", "Acknowledged")
	execNoErr(t, db, `
		INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, pending_encrypted_password, status, operation_type, pending_set_command_uuid)
		VALUES ('tracked', ?, ?, 'pending', 'install', 'set-10')`, old, rotated)
	execNoErr(t, db, `
		INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, status, operation_type, set_command_uuid, verify_command_uuid)
		VALUES ('verified', ?, 'verified', 'install', 'set-x', 'verify-x')`, old)
	queue("deleted", "set-11", "SetRecoveryLock", "2026-09-01 10:00:00", "Acknowledged")
	execNoErr(t, db, `
		INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, pending_encrypted_password, status, operation_type, deleted)
		VALUES ('deleted', ?, ?, 'pending', 'install', 1)`, old, rotated)

	applyNext(t, db)

	type row struct {
		Status            sql.NullString `db:"status"`
		Current           []byte         `db:"encrypted_password"`
		Pending           []byte         `db:"pending_encrypted_password"`
		PendingSetCommand sql.NullString `db:"pending_set_command_uuid"`
		PendingVerify     sql.NullString `db:"pending_verify_command_uuid"`
		SetCommand        sql.NullString `db:"set_command_uuid"`
		VerifyCommand     sql.NullString `db:"verify_command_uuid"`
	}
	get := func(id string) row {
		var r row
		require.NoError(t, db.Get(&r, `
			SELECT status, encrypted_password, pending_encrypted_password, pending_set_command_uuid,
				pending_verify_command_uuid, set_command_uuid, verify_command_uuid
			FROM host_recovery_key_passwords WHERE host_uuid = ?`, id))
		return r
	}
	str := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	pending, verifying := str("pending"), str("verifying")

	// unanswered: waits on its command, and a first set's password is staged where the verify reads it
	require.Equal(t, row{Status: pending, Current: old, Pending: rotated, PendingSetCommand: str("set-1")}, get("rotation-unanswered"))
	require.Equal(t, row{Status: pending, Current: rotated, Pending: rotated, PendingSetCommand: str("set-2")}, get("first-notnow"))

	// acknowledged: verifying with no verify command yet, for the cron to send
	require.Equal(t, row{Status: verifying, Current: old, Pending: rotated, SetCommand: str("set-3")}, get("rotation-acked"))
	require.Equal(t, row{Status: verifying, Current: rotated, Pending: rotated, SetCommand: str("set-4")}, get("first-acked"))
	require.Equal(t, row{Status: verifying, Current: old, SetCommand: str("set-5")}, get("clear-acked"))

	// rejected: back to the cron; a first set drops the password the device never took
	require.Equal(t, row{}, get("first-error"))
	require.Equal(t, row{Current: old, Pending: rotated}, get("rotation-error"))

	// the command cleanup removed the acknowledged command: verified like an acknowledged one
	require.Equal(t, row{Status: verifying, Current: old, Pending: rotated}, get("no-command"))

	require.Equal(t, row{Status: pending, Current: old, Pending: rotated, PendingSetCommand: str("set-9")}, get("two-sets"))

	require.Equal(t, row{Status: pending, Current: old, Pending: rotated, PendingSetCommand: str("set-10")}, get("tracked"))
	require.Equal(t, row{Status: str("verified"), Current: old, SetCommand: str("set-x"), VerifyCommand: str("verify-x")}, get("verified"))
	require.Equal(t, row{Status: pending, Current: old, Pending: rotated}, get("deleted"))

	// every repaired pending password is archived, since the device may hold it
	type archived struct {
		HostUUID   string         `db:"host_uuid"`
		Password   []byte         `db:"encrypted_password"`
		SetCommand sql.NullString `db:"set_command_uuid"`
	}
	var archive []archived
	require.NoError(t, db.Select(&archive, `
		SELECT host_uuid, encrypted_password, set_command_uuid FROM host_recovery_key_password_archive ORDER BY host_uuid`))
	require.Equal(t, []archived{
		{HostUUID: "no-command", Password: rotated},
		{HostUUID: "rotation-acked", Password: rotated, SetCommand: str("set-3")},
		{HostUUID: "rotation-error", Password: rotated, SetCommand: str("set-7")},
		{HostUUID: "rotation-unanswered", Password: rotated, SetCommand: str("set-1")},
		{HostUUID: "two-sets", Password: rotated, SetCommand: str("set-9")},
	}, archive)
}
