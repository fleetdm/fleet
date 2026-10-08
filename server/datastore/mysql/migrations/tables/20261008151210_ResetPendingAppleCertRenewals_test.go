package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261008151210(t *testing.T) {
	db := applyUpToPrev(t)

	const plist = `<?xml version="1.0"?><plist/>`
	enroll := func(id string) {
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, id)
		execNoErr(t, db, `
			INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
			VALUES (?, ?, 'Device', 'com.apple.mgmt.test', 'magic', 'abcdef', 1)`, id, id)
	}
	queueCommand := func(cmdUUID string, hostIDs ...string) {
		execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command, name) VALUES (?, 'InstallProfile', ?, '')`, cmdUUID, plist)
		for _, id := range hostIDs {
			execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`, id, cmdUUID)
		}
	}
	associate := func(id string, renewCmdUUID *string) {
		execNoErr(t, db, `INSERT INTO nano_cert_auth_associations (id, sha256, renew_command_uuid) VALUES (?, ?, ?)`,
			id, fmt.Sprintf("%064s", id), renewCmdUUID)
	}

	for _, id := range []string{"host-a", "host-b", "host-c"} {
		enroll(id)
	}
	// a bucketed renewal shared by two hosts, and an unrelated command
	queueCommand("renewal", "host-a", "host-b")
	queueCommand("other", "host-a", "host-c")
	associate("host-a", new("renewal"))
	associate("host-b", new("renewal"))
	associate("host-c", nil)

	applyNext(t, db)

	var pending int
	require.NoError(t, db.Get(&pending, `SELECT COUNT(*) FROM nano_cert_auth_associations WHERE renew_command_uuid IS NOT NULL`))
	require.Zero(t, pending)

	type queueRow struct {
		ID          string `db:"id"`
		CommandUUID string `db:"command_uuid"`
		Active      bool   `db:"active"`
	}
	var queue []queueRow
	require.NoError(t, db.Select(&queue, `SELECT id, command_uuid, active FROM nano_enrollment_queue ORDER BY command_uuid, id`))
	require.Equal(t, []queueRow{
		{"host-a", "other", true},
		{"host-c", "other", true},
		{"host-a", "renewal", false},
		{"host-b", "renewal", false},
	}, queue)
}
