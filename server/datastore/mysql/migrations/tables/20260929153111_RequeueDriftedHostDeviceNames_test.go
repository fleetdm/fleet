package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260929153111(t *testing.T) {
	db := applyUpToPrev(t)

	const driftDetail = "Host was renamed on the device and no longer matches the fleet's naming template."

	insertHost := func(uuid, platform string, mdmEnrolled, personal bool) {
		hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
			"oh-"+uuid, "nk-"+uuid, uuid+".local", uuid, platform)
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, is_personal_enrollment) VALUES (?, ?, 'https://fleet.local', ?)`,
			hostID, mdmEnrolled, personal)
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, uuid)
		execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled) VALUES (?, ?, 'Device', 'topic', 'magic', 'token', ?)`,
			uuid, uuid, mdmEnrolled)
	}
	insertRow := func(uuid string, status *string, commandUUID, detail string, retries int) {
		execNoErr(t, db, `INSERT INTO host_mdm_apple_device_names (host_uuid, status, command_uuid, expected_device_name, detail, retries) VALUES (?, ?, ?, 'WS-1', ?, ?)`,
			uuid, status, commandUUID, detail, retries)
	}
	failed, verified := "failed", "verified"

	insertHost("drifted", "darwin", true, false)
	insertRow("drifted", &failed, "DEVNAME-1", driftDetail, 2)

	insertHost("drifted-ios", "ios", true, false)
	insertRow("drifted-ios", &failed, "DEVNAME-2", driftDetail, 0)

	insertHost("drifted-unenrolled", "darwin", false, false)
	insertRow("drifted-unenrolled", &failed, "DEVNAME-3", driftDetail, 0)

	insertHost("drifted-byod", "ios", true, true)
	insertRow("drifted-byod", &failed, "DEVNAME-4", driftDetail, 0)

	insertHost("rejected", "ios", true, false)
	insertRow("rejected", &failed, "DEVNAME-5", "MCMDMErrorDomain (12026): The device is not supervised.", 3)

	insertHost("verified", "darwin", true, false)
	insertRow("verified", &verified, "DEVNAME-6", "", 0)

	applyNext(t, db)

	type row struct {
		Status      sql.NullString `db:"status"`
		CommandUUID sql.NullString `db:"command_uuid"`
		Retries     int            `db:"retries"`
	}
	get := func(uuid string) row {
		var r row
		require.NoError(t, db.Get(&r, `SELECT status, command_uuid, retries FROM host_mdm_apple_device_names WHERE host_uuid = ?`, uuid))
		return r
	}

	for _, uuid := range []string{"drifted", "drifted-ios"} {
		r := get(uuid)
		require.False(t, r.Status.Valid, uuid)
		require.False(t, r.CommandUUID.Valid, uuid)
		require.Zero(t, r.Retries, uuid)
	}

	for uuid, want := range map[string]row{
		"drifted-unenrolled": {Status: validString("failed"), CommandUUID: validString("DEVNAME-3")},
		"drifted-byod":       {Status: validString("failed"), CommandUUID: validString("DEVNAME-4")},
		"rejected":           {Status: validString("failed"), CommandUUID: validString("DEVNAME-5"), Retries: 3},
		"verified":           {Status: validString("verified"), CommandUUID: validString("DEVNAME-6")},
	} {
		require.Equal(t, want, get(uuid), uuid)
	}
}
