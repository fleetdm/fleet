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
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestDiskEncryption(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestGetHostArchivedDiskEncryptionKey", testGetHostArchivedDiskEncryptionKey},
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

func testGetHostArchivedDiskEncryptionKey(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	newHost := func(suffix, serial string, teamID *uint) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new("archived-" + suffix),
			UUID:            "archived-" + suffix,
			Hostname:        "archived-" + suffix,
			HardwareSerial:  serial,
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	archiveRow := func(hostID uint, serial, key string, createdAt time.Time) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys_archive (host_id, hardware_serial, base64_encrypted, base64_encrypted_salt, key_slot, created_at)
VALUES (?, ?, ?, ?, NULL, ?)`, hostID, serial, base64.StdEncoding.EncodeToString([]byte(key)), "", createdAt)
			return err
		})
	}

	const serial = "SHAREDSERIAL1"
	now := time.Now().UTC().Truncate(time.Second)

	victimTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "archived-victim-fleet"})
	require.NoError(t, err)
	claimantTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "archived-claimant-fleet"})
	require.NoError(t, err)

	victim := newHost("victim", serial, &victimTeam.ID)
	// Two rows for the victim so the "newest wins" ordering is actually exercised.
	archiveRow(victim.ID, serial, "older-key", now.Add(-2*time.Hour))
	archiveRow(victim.ID, serial, "newest-key", now.Add(-1*time.Hour))

	// A second host carrying the same serial in another fleet, with no archived row
	// of its own. This is the shape the cross-fleet disclosure relied on.
	claimant := newHost("claimant", serial, &claimantTeam.ID)

	t.Run("host id match ignores the fallback flag", func(t *testing.T) {
		for _, fallback := range []bool{false, true} {
			key, err := ds.GetHostArchivedDiskEncryptionKey(ctx, victim, fallback)
			require.NoError(t, err)
			requireDecodes(t, "newest-key", key.Base64Encrypted)
		}
	})

	t.Run("no fallback means no serial lookup", func(t *testing.T) {
		_, err := ds.GetHostArchivedDiskEncryptionKey(ctx, claimant, false)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("fallback finds the newest row for the serial", func(t *testing.T) {
		key, err := ds.GetHostArchivedDiskEncryptionKey(ctx, claimant, true)
		require.NoError(t, err)
		requireDecodes(t, "newest-key", key.Base64Encrypted)
		require.Equal(t, victim.ID, key.HostID, "the row returned belongs to the other fleet's host")
	})

	t.Run("fallback needs a serial to match on", func(t *testing.T) {
		noSerial := newHost("noserial", "", &claimantTeam.ID)
		_, err := ds.GetHostArchivedDiskEncryptionKey(ctx, noSerial, true)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})
}

func requireDecodes(t *testing.T, want, gotBase64 string) {
	t.Helper()
	got, err := base64.StdEncoding.DecodeString(gotBase64)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}
