package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261103181933(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(name, platform string) int64 {
		res, err := db.Exec(`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
			name, name, name, name, platform)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	macID := insertHost("mac", "darwin")
	winID := insertHost("win", "windows")
	const deletedHostID = 999999

	for _, hostID := range []int64{macID, winID, deletedHostID} {
		execNoErr(t, db, `INSERT INTO microsoft_compliance_partner_host_statuses (host_id, device_id, user_principal_name, managed, compliant) VALUES (?, ?, ?, 1, 1)`,
			hostID, "device", "user@example.com")
	}

	applyNext(t, db)

	var hostIDs []int64
	require.NoError(t, db.Select(&hostIDs, `SELECT host_id FROM microsoft_compliance_partner_host_statuses ORDER BY host_id`))
	// Both remaining rows share a device ID, so the new index must not be unique.
	require.Equal(t, []int64{winID, deletedHostID}, hostIDs)
	require.True(t, indexExists(db, "microsoft_compliance_partner_host_statuses", "idx_microsoft_compliance_partner_host_statuses_device_id"))
}
