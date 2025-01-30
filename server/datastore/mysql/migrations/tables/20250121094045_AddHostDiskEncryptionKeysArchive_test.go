package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20250121094045(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up: 2 hosts and 3 keys
	i := uint(1)
	newHost := func(platform string) uint {
		id := fmt.Sprintf("%d", i)
		i++
		hostID := uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO hosts (hardware_serial, osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
			id, id, id, id, platform,
		))
		return hostID
	}
	ubuntuHostID := newHost("ubuntu")
	macOSHostID := newHost("darwin")

	hostIDs := []uint{ubuntuHostID, macOSHostID, 9999}
	for _, hostID := range hostIDs {
		execNoErr(t, db,
			`INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, base64_encrypted_salt, key_slot) VALUES (?, ?, ?, ?)`,
			hostID, fmt.Sprintf("encrypted-%d", hostID), "salt", 1,
		)
	}
	timeBeforeMigration := time.Now().Add(-1 * time.Second) // allow for the server and DB time to be off by 1 second

	// Apply current migration.
	applyNext(t, db)

	type archiveKey struct {
		HostID          uint      `db:"host_id"`
		HardwareSerial  string    `db:"hardware_serial"`
		Base64Encrypted string    `db:"base64_encrypted"`
		CreatedAt       time.Time `db:"created_at"`
	}
	var keys []archiveKey
	require.NoError(t,
		db.Select(&keys,
			`SELECT host_id, hardware_serial, base64_encrypted, created_at FROM host_disk_encryption_keys_archive ORDER BY host_id ASC`))
	require.Len(t, keys, 3)
	for i := range 3 {
		require.Equal(t, hostIDs[i], keys[i].HostID)
		require.Equal(t, fmt.Sprintf("encrypted-%d", hostIDs[i]), keys[i].Base64Encrypted)
		require.GreaterOrEqual(t, keys[i].CreatedAt.Unix(), timeBeforeMigration.Unix())
	}
	require.Equal(t, "1", keys[0].HardwareSerial)
	require.Equal(t, "2", keys[1].HardwareSerial)
	require.Equal(t, "", keys[2].HardwareSerial)

}
