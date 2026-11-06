package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261110121303(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(name, uuid string) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'ios')`, name, name, name, uuid)
	}
	bound := insertHost("bound", "uuid-bound")
	registered := insertHost("registered", "uuid-registered")
	insertHost("unbound", "uuid-unbound")
	noUUID := insertHost("no-uuid", "")

	execNoErr(t, db, `INSERT INTO host_google_device_ids (host_id, google_device_id) VALUES (?, 'device-bound'), (?, 'device-registered'), (?, 'device-no-uuid')`,
		bound, registered, noUUID)
	// A host that already signed in keeps its row and gains the ID.
	execNoErr(t, db, `INSERT INTO google_conditional_access_devices (host_uuid, customer_id, registration_attempts) VALUES ('uuid-registered', 'C01', 2)`)
	execNoErr(t, db, `INSERT INTO google_conditional_access_device_users (host_uuid, device_user_name, state)
		VALUES ('uuid-registered', 'devices/device-registered/deviceUsers/a', 'managed')`)

	applyNext(t, db)

	type row struct {
		HostUUID             string  `db:"host_uuid"`
		CustomerID           string  `db:"customer_id"`
		GoogleDeviceID       *string `db:"google_device_id"`
		RegistrationAttempts uint    `db:"registration_attempts"`
	}
	var rows []row
	require.NoError(t, db.Select(&rows, `SELECT host_uuid, customer_id, google_device_id, registration_attempts
		FROM google_conditional_access_devices ORDER BY host_uuid`))
	require.Equal(t, []row{
		{HostUUID: "uuid-bound", CustomerID: "", GoogleDeviceID: new("device-bound")},
		{HostUUID: "uuid-registered", CustomerID: "C01", GoogleDeviceID: new("device-registered"), RegistrationAttempts: 2},
	}, rows)

	var names int
	require.NoError(t, db.Get(&names, `SELECT COUNT(*) FROM google_conditional_access_device_users`))
	require.Equal(t, 1, names)

	var oldTables int
	require.NoError(t, db.Get(&oldTables, `SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = 'host_google_device_ids'`))
	require.Zero(t, oldTables)

	// The ID is unique across hosts, and a host can have none.
	execNoErr(t, db, `INSERT INTO google_conditional_access_devices (host_uuid, customer_id) VALUES ('uuid-unbound', 'C01'), ('uuid-other', 'C01')`)
	_, err := db.Exec(`UPDATE google_conditional_access_devices SET google_device_id = 'DEVICE-BOUND' WHERE host_uuid = 'uuid-unbound'`)
	require.ErrorContains(t, err, "Duplicate entry")
}
