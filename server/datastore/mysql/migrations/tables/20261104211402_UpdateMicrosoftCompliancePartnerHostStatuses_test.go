package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261104211402(t *testing.T) {
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
	win1ID := insertHost("win1", "windows")
	win2ID := insertHost("win2", "windows")
	win3ID := insertHost("win3", "windows")
	const deletedHostID = 999999

	for hostID, deviceID := range map[int64]string{
		macID:         "mac-device",
		win1ID:        "shared-device",
		win2ID:        "shared-device",
		win3ID:        "win3-device",
		deletedHostID: "orphan-device",
	} {
		execNoErr(t, db, `INSERT INTO microsoft_compliance_partner_host_statuses (host_id, device_id, user_principal_name, managed, compliant) VALUES (?, ?, ?, 1, 1)`,
			hostID, deviceID, "user@example.com")
	}

	applyNext(t, db)

	// macOS rows and device IDs shared by several hosts are deleted.
	var rows []struct {
		HostID   int64  `db:"host_id"`
		DeviceID string `db:"device_id"`
		Verified bool   `db:"verified"`
	}
	require.NoError(t, db.Select(&rows, `SELECT host_id, device_id, verified FROM microsoft_compliance_partner_host_statuses ORDER BY host_id`))
	require.Len(t, rows, 2)
	require.Equal(t, win3ID, rows[0].HostID)
	require.Equal(t, "win3-device", rows[0].DeviceID)
	require.False(t, rows[0].Verified)
	require.Equal(t, int64(deletedHostID), rows[1].HostID)
	require.False(t, rows[1].Verified)
	require.True(t, indexExists(db, "microsoft_compliance_partner_host_statuses", "idx_microsoft_compliance_partner_host_statuses_device_id"))
}
