package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261002135412(t *testing.T) {
	db := applyUpToPrev(t)

	const driftDetail = "Host was renamed on the device and no longer matches the fleet's naming template."

	insertHost := func(uuid string, mdmEnrolled bool) {
		hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'darwin')`,
			"oh-"+uuid, "nk-"+uuid, uuid+".local", uuid)
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url) VALUES (?, ?, 'https://fleet.local')`, hostID, mdmEnrolled)
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, uuid)
		execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled) VALUES (?, ?, 'Device', 'topic', 'magic', 'token', ?)`,
			uuid, uuid, mdmEnrolled)
	}
	insertFailedRow := func(uuid, commandUUID, detail string) {
		execNoErr(t, db, `INSERT INTO host_mdm_apple_device_names (host_uuid, status, command_uuid, expected_device_name, detail) VALUES (?, 'failed', ?, 'WS-1', ?)`,
			uuid, commandUUID, detail)
	}

	insertHost("drifted", true)
	insertFailedRow("drifted", "DEVNAME-1", driftDetail)

	insertHost("drifted-unenrolled", false)
	insertFailedRow("drifted-unenrolled", "DEVNAME-2", driftDetail)

	insertHost("rejected", true)
	insertFailedRow("rejected", "DEVNAME-3", "MCMDMErrorDomain (12026): The device is not supervised.")

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

	require.Equal(t, row{}, get("drifted"))
	require.Equal(t, row{Status: validString("failed"), CommandUUID: validString("DEVNAME-2")}, get("drifted-unenrolled"))
	require.Equal(t, row{Status: validString("failed"), CommandUUID: validString("DEVNAME-3")}, get("rejected"))
}
