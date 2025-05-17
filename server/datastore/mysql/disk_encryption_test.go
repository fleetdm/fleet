package mysql

import (
	"context"
	"encoding/base64"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiskEncryption(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestCleanupDiskEncryptionKeysOnTeamChange", testCleanupDiskEncryptionKeysOnTeamChange},
		{"TestDeleteLUKSData", testDeleteLUKSData},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testCleanupDiskEncryptionKeysOnTeamChange(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// No-op test
	assert.NoError(t, ds.CleanupDiskEncryptionKeysOnTeamChange(ctx, []uint{1, 2, 3}, nil))
}

func testDeleteLUKSData(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hostOne, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	hostTwo, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo.local-zzz",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
	})
	require.NoError(t, err)

	// Add a LUKS user key
	randomBits := base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))
	var keySlot uint = 1

	err = ds.SaveLUKSData(ctx, hostOne, randomBits, randomBits, keySlot)
	require.NoError(t, err)

	// Try to delete a non-existent LUKS key
	err = ds.DeleteLUKSData(ctx, hostTwo.ID, keySlot)
	require.NoError(t, err)

	// Try to delete the wrong key slot
	err = ds.DeleteLUKSData(ctx, hostOne.ID, keySlot+1)
	require.NoError(t, err)

	err = ds.DeleteLUKSData(ctx, hostOne.ID, keySlot)
	require.NoError(t, err)

	_, err = ds.GetHostDiskEncryptionKey(ctx, hostOne.ID)
	require.True(t, fleet.IsNotFound(err))
}
