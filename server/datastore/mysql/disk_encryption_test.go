package mysql

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
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

func TestBitLockerPINRequests(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Lifecycle", testBitLockerPINRequestLifecycle},
		{"TakeIsOnlyEverOnce", testBitLockerPINRequestTakeIsOnlyEverOnce},
		{"ExpiredIsNotCollectable", testBitLockerPINRequestExpiredIsNotCollectable},
		{"ResubmitReplaces", testBitLockerPINRequestResubmitReplaces},
		{"Delete", testBitLockerPINRequestDelete},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

// newBitLockerPINHost creates a host with a Windows MDM enrollment, which is what carries the denormalized
// bitlocker_pin_request_pending flag the orbit config poll reads.
func newBitLockerPINHost(t *testing.T, ds *Datastore) *fleet.Host {
	t.Helper()
	ctx := context.Background()

	hostUUID := uuid.New().String()
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "pin-host-" + hostUUID[:8],
		OsqueryHostID: new(hostUUID),
		NodeKey:       new(hostUUID),
		UUID:          hostUUID,
		Platform:      "windows",
	})
	require.NoError(t, err)

	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-PIN",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		HostUUID:               host.UUID,
	}))

	return host
}

// pendingFlag reads the denormalized flag straight from the enrollment row, which is the value the orbit config poll
// actually gates on.
func pendingFlag(t *testing.T, ds *Datastore, hostUUID string) bool {
	t.Helper()
	var pending bool
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &pending,
			`SELECT bitlocker_pin_request_pending FROM mdm_windows_enrollments WHERE host_uuid = ?`, hostUUID)
	})
	return pending
}

func testBitLockerPINRequestLifecycle(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newBitLockerPINHost(t, ds)

	// Nothing submitted yet.
	_, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, pendingFlag(t, ds, host.UUID))

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))

	req, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestPending, req.Status)
	require.Empty(t, req.Error)
	// Queuing and raising the flag commit together, so the poll can see it immediately.
	require.True(t, pendingFlag(t, ds, host.UUID))

	pin, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "encrypted-pin", pin)
	// Collected, so there is nothing left to wake the agent for.
	require.False(t, pendingFlag(t, ds, host.UUID))

	req, err = ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestDelivered, req.Status)

	require.NoError(t, ds.SetBitLockerPINRequestOutcome(ctx, host, fleet.BitLockerPINRequestSet, ""))
	req, err = ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	// The row survives success so the waiting page has a positive signal to poll for.
	require.Equal(t, fleet.BitLockerPINRequestSet, req.Status)
	require.False(t, pendingFlag(t, ds, host.UUID))
}

func testBitLockerPINRequestTakeIsOnlyEverOnce(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))

	pin, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "encrypted-pin", pin)

	// A replayed or concurrent collect must come away with nothing, and must not resurrect the ciphertext.
	_, err = ds.TakeBitLockerPINRequest(ctx, host)
	require.True(t, fleet.IsNotFound(err))

	var stored *string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &stored,
			`SELECT pin_encrypted FROM host_bitlocker_pin_requests WHERE host_id = ?`, host.ID)
	})
	require.Nil(t, stored)
}

func testBitLockerPINRequestExpiredIsNotCollectable(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))

	// Age the submission past its TTL on the database clock, which is what the collect query compares against.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
UPDATE host_bitlocker_pin_requests SET created_at = DATE_SUB(NOW(6), INTERVAL ? SECOND) WHERE host_id = ?`,
			int(fleet.BitLockerPINRequestTTL.Seconds())+60, host.ID)
		return err
	})

	_, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.True(t, fleet.IsNotFound(err))
	// A collect that finds nothing clears the flag rather than leaving it to wake the agent on every poll.
	require.False(t, pendingFlag(t, ds, host.UUID))
}

func testBitLockerPINRequestResubmitReplaces(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "first-pin"))
	require.NoError(t, ds.SetBitLockerPINRequestOutcome(ctx, host, fleet.BitLockerPINRequestFailed, "PIN rejected"))

	req, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestFailed, req.Status)
	require.Equal(t, "PIN rejected", req.Error)
	require.False(t, pendingFlag(t, ds, host.UUID))

	// Retrying supersedes the failure: one row per host, back to pending, with the error cleared.
	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "second-pin"))
	req, err = ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestPending, req.Status)
	require.Empty(t, req.Error)
	require.True(t, pendingFlag(t, ds, host.UUID))

	pin, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "second-pin", pin)
}

func testBitLockerPINRequestDelete(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))
	require.True(t, pendingFlag(t, ds, host.UUID))

	require.NoError(t, ds.DeleteBitLockerPINRequest(ctx, host))

	_, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, pendingFlag(t, ds, host.UUID))
}
