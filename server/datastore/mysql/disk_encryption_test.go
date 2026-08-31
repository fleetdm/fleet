package mysql

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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

	newHostWithKey := func(t *testing.T, suffix, platform string, teamID *uint) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new("key-cleanup-" + suffix),
			UUID:            "key-cleanup-" + suffix,
			Hostname:        "key-cleanup-" + suffix,
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, h, base64.StdEncoding.EncodeToString([]byte("k")), "", new(true))
		require.NoError(t, err)
		_, err = ds.GetHostDiskEncryptionKey(ctx, h.ID)
		require.NoError(t, err, "precondition: the host has a key")
		return h
	}

	hasKey := func(t *testing.T, hostID uint) bool {
		_, err := ds.GetHostDiskEncryptionKey(ctx, hostID)
		if err != nil && fleet.IsNotFound(err) {
			return false
		}
		require.NoError(t, err)
		return true
	}

	// Each case moves one host of each platform into a fleet with the given
	// settings. A key survives only when its own platform still escrows to
	// Fleet there — before per-platform settings this was decided for every
	// platform by whether the fleet had an Apple FileVault profile.
	for i, tc := range []struct {
		name                               string
		macOSEnforce, macOSEscrow          bool
		windowsEnabled, linuxEscrow        bool
		keepDarwin, keepWindows, keepLinux bool
	}{
		{
			name:         "everything on keeps every key",
			macOSEnforce: true, macOSEscrow: true, windowsEnabled: true, linuxEscrow: true,
			keepDarwin: true, keepWindows: true, keepLinux: true,
		},
		{
			name:       "everything off deletes every key",
			keepDarwin: false, keepWindows: false, keepLinux: false,
		},
		{
			name:         "macOS enforcement without escrow drops the macOS key",
			macOSEnforce: true, windowsEnabled: true, linuxEscrow: true,
			keepDarwin: false, keepWindows: true, keepLinux: true,
		},
		{
			name:        "macOS escrow alone keeps only the macOS key",
			macOSEscrow: true,
			keepDarwin:  true, keepWindows: false, keepLinux: false,
		},
		{
			name:           "Windows only keeps only the Windows key",
			windowsEnabled: true,
			keepDarwin:     false, keepWindows: true, keepLinux: false,
		},
		{
			name:        "Linux only keeps only the Linux key",
			linuxEscrow: true,
			keepDarwin:  false, keepWindows: false, keepLinux: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tm, err := ds.NewTeam(ctx, &fleet.Team{
				Name: fmt.Sprintf("key-cleanup-fleet-%d", i),
				Config: fleet.TeamConfig{MDM: fleet.TeamMDM{
					MacOSSettings: fleet.MacOSSettings{
						EnableDiskEncryption:          optjson.SetBool(tc.macOSEnforce),
						EnableEscrowDiskEncryptionKey: optjson.SetBool(tc.macOSEscrow),
					},
					WindowsSettings: fleet.WindowsSettings{
						EnableDiskEncryption: optjson.SetBool(tc.windowsEnabled),
					},
					LinuxSettings: fleet.LinuxSettings{
						EnableEscrowDiskEncryptionKey: optjson.SetBool(tc.linuxEscrow),
					},
				}},
			})
			require.NoError(t, err)

			suffix := fmt.Sprintf("%d", i)
			darwin := newHostWithKey(t, "darwin-"+suffix, "darwin", &tm.ID)
			windows := newHostWithKey(t, "windows-"+suffix, "windows", &tm.ID)
			linux := newHostWithKey(t, "linux-"+suffix, "ubuntu", &tm.ID)

			require.NoError(t, ds.CleanupDiskEncryptionKeysOnTeamChange(ctx,
				[]uint{darwin.ID, windows.ID, linux.ID}, &tm.ID))

			require.Equal(t, tc.keepDarwin, hasKey(t, darwin.ID), "darwin key")
			require.Equal(t, tc.keepWindows, hasKey(t, windows.ID), "windows key")
			require.Equal(t, tc.keepLinux, hasKey(t, linux.ID), "linux key")
		})
	}
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

	_, err = ds.SaveLUKSData(ctx, hostOne, randomBits, randomBits, &keySlot)
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
