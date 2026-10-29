package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261029172033(t *testing.T) {
	db := applyUpToPrev(t)

	// insert two Android hosts with a device and a macOS host without one
	firstHostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES ('host1', 'host1', 'host1', 'host1', 'android')`)
	execNoErr(t, db, `INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, 'device1', 'host1')`, firstHostID)
	secondHostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES ('host2', 'host2', 'host2', 'host2', 'android')`)
	execNoErr(t, db, `INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, 'device2', 'host2')`, secondHostID)
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES ('host3', 'host3', 'host3', 'host3', 'darwin')`)

	applyNext(t, db)

	// read the rows of host_android_software_installs, each Android host should have one pending row for an app that's
	// never in scope, and the macOS host none
	var installs []struct {
		HostID      int64  `db:"host_id"`
		AdamID      string `db:"adam_id"`
		InstallType string `db:"install_type"`
		Status      string `db:"status"`
		NeedsResend bool   `db:"needs_resend"`
	}
	err := db.Select(&installs, `SELECT host_id, adam_id, install_type, status, needs_resend FROM host_android_software_installs ORDER BY host_id`)
	require.NoError(t, err)
	require.Len(t, installs, 2)
	require.Equal(t, firstHostID, installs[0].HostID)
	require.Equal(t, "TEMPORARY_FOR_MIGRATION", installs[0].AdamID)
	require.Equal(t, "AVAILABLE", installs[0].InstallType)
	require.Equal(t, "pending", installs[0].Status)
	require.False(t, installs[0].NeedsResend)
	require.Equal(t, secondHostID, installs[1].HostID)
	require.Equal(t, "TEMPORARY_FOR_MIGRATION", installs[1].AdamID)
}
