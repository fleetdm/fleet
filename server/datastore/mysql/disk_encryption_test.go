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
		{"TestCleanupDiskEncryptionKeysOnTeamChange", testCleanupDiskEncryptionKeysOnTeamChange},
		{"TestDeleteLUKSData", testDeleteLUKSData},
		{"TestBitLockerPINRequestLifecycle", testBitLockerPINRequestLifecycle},
		{"TestBitLockerPINRequestExpiredIsNotCollectable", testBitLockerPINRequestExpiredIsNotCollectable},
		{"TestBitLockerPINRequestResubmitReplaces", testBitLockerPINRequestResubmitReplaces},
		{"TestBitLockerPINRequestDelete", testBitLockerPINRequestDelete},
		{"TestBitLockerPINRequestCleanup", testBitLockerPINRequestCleanup},
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

// newBitLockerPINHost creates a host with a Windows MDM enrollment, which carries the pending flag the orbit config
// poll reads.
func newBitLockerPINHost(t *testing.T, ds *Datastore) *fleet.Host {
	t.Helper()
	hostUUID := uuid.NewString()
	host, err := ds.NewHost(t.Context(), &fleet.Host{
		Hostname:      "pin-host-" + hostUUID,
		OsqueryHostID: new(hostUUID),
		NodeKey:       new(hostUUID),
		UUID:          hostUUID,
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, host)
	return host
}

// bitLockerPINPending reads the pending flag the way the orbit config poll does.
func bitLockerPINPending(t *testing.T, ds *Datastore, hostUUID string) bool {
	t.Helper()
	state, err := ds.GetMDMWindowsHostConfigState(t.Context(), hostUUID)
	require.NoError(t, err)
	return state.BitLockerPINRequestPending
}

// storedBitLockerPIN returns the stored ciphertext, which must be NULL once nothing can collect it.
func storedBitLockerPIN(t *testing.T, ds *Datastore, hostID uint) *string {
	t.Helper()
	var stored *string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &stored, `SELECT pin_encrypted FROM host_bitlocker_pin_requests WHERE host_id = ?`, hostID)
	})
	return stored
}

// ageBitLockerPINRequest moves a timestamp column back by the given duration, on the database clock the queries
// compare against. An explicit value wins over ON UPDATE CURRENT_TIMESTAMP, so the row does not snap back.
func ageBitLockerPINRequest(t *testing.T, ds *Datastore, hostID uint, column string, by time.Duration) {
	t.Helper()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), fmt.Sprintf(
			`UPDATE host_bitlocker_pin_requests SET %s = DATE_SUB(NOW(6), INTERVAL ? SECOND) WHERE host_id = ?`, column),
			int(by.Seconds()), hostID)
		return err
	})
}

func testBitLockerPINRequestLifecycle(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newBitLockerPINHost(t, ds)

	_, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, bitLockerPINPending(t, ds, host.UUID))

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))
	req, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestPending, req.Status)
	// Queuing and raising the flag commit together, so the poll can see it immediately.
	require.True(t, bitLockerPINPending(t, ds, host.UUID))

	pin, requestUUID, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "encrypted-pin", pin)
	require.False(t, bitLockerPINPending(t, ds, host.UUID))
	req, err = ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestDelivered, req.Status)

	// A replayed or concurrent collect comes away with nothing, and the ciphertext is already gone.
	_, _, err = ds.TakeBitLockerPINRequest(ctx, host)
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, storedBitLockerPIN(t, ds, host.ID))

	// An outcome naming a submission this host never collected is refused, so a host cannot mark itself as having a PIN.
	// A well-formed id that matches nothing and one that does not parse are refused the same way.
	for _, forged := range []string{uuid.NewString(), "not-the-collected-request"} {
		require.True(t, fleet.IsNotFound(ds.SetBitLockerPINRequestOutcome(ctx, host, forged, fleet.BitLockerPINRequestSet, "")))
	}

	require.NoError(t, ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID, fleet.BitLockerPINRequestSet, ""))
	req, err = ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	// The row survives success so the waiting page has a positive signal to poll for.
	require.Equal(t, fleet.BitLockerPINRequestSet, req.Status)
}

func testBitLockerPINRequestExpiredIsNotCollectable(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))
	ageBitLockerPINRequest(t, ds, host.ID, "created_at", fleet.BitLockerPINRequestTTL+time.Minute)

	_, _, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.True(t, fleet.IsNotFound(err))
	// A collect that finds nothing clears the flag rather than leaving it to wake the agent on every poll.
	require.False(t, bitLockerPINPending(t, ds, host.UUID))
}

func testBitLockerPINRequestResubmitReplaces(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "first-pin"))
	_, firstUUID, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.NoError(t, err)
	require.NoError(t, ds.SetBitLockerPINRequestOutcome(ctx, host, firstUUID, fleet.BitLockerPINRequestFailed, "PIN rejected"))
	req, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestFailed, req.Status)
	require.Equal(t, "PIN rejected", req.Error)

	// Retrying supersedes the failure: one row per host, back to pending, with the error cleared.
	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "second-pin"))
	req, err = ds.GetBitLockerPINRequest(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestPending, req.Status)
	require.Empty(t, req.Error)
	require.True(t, bitLockerPINPending(t, ds, host.UUID))

	pin, secondUUID, err := ds.TakeBitLockerPINRequest(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "second-pin", pin)

	// Both requests are now past delivery, so only the request id stops a late outcome for the first landing on the second.
	require.True(t, fleet.IsNotFound(ds.SetBitLockerPINRequestOutcome(ctx, host, firstUUID, fleet.BitLockerPINRequestSet, "")))
	require.NoError(t, ds.SetBitLockerPINRequestOutcome(ctx, host, secondUUID, fleet.BitLockerPINRequestSet, ""))
}

func testBitLockerPINRequestDelete(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newBitLockerPINHost(t, ds)

	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))
	require.True(t, bitLockerPINPending(t, ds, host.UUID))

	require.NoError(t, ds.DeleteBitLockerPINRequest(ctx, host))

	_, err := ds.GetBitLockerPINRequest(ctx, host.ID)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, bitLockerPINPending(t, ds, host.UUID))
}

func testBitLockerPINRequestCleanup(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	settle := func(t *testing.T, host *fleet.Host, outcome fleet.BitLockerPINRequestStatus, reason string) {
		t.Helper()
		require.NoError(t, ds.QueueBitLockerPINRequest(ctx, host, "encrypted-pin"))
		_, requestUUID, err := ds.TakeBitLockerPINRequest(ctx, host)
		require.NoError(t, err)
		require.NoError(t, ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID, outcome, reason))
	}
	exists := func(t *testing.T, hostID uint) bool {
		t.Helper()
		_, err := ds.GetBitLockerPINRequest(ctx, hostID)
		if fleet.IsNotFound(err) {
			return false
		}
		require.NoError(t, err)
		return true
	}
	pastRetention := bitLockerPINRequestRetention + time.Hour

	oldSet := newBitLockerPINHost(t, ds)
	settle(t, oldSet, fleet.BitLockerPINRequestSet, "")
	ageBitLockerPINRequest(t, ds, oldSet.ID, "updated_at", pastRetention)

	oldFailed := newBitLockerPINHost(t, ds)
	settle(t, oldFailed, fleet.BitLockerPINRequestFailed, "PIN rejected")
	ageBitLockerPINRequest(t, ds, oldFailed.ID, "updated_at", pastRetention)

	recentFailed := newBitLockerPINHost(t, ds)
	settle(t, recentFailed, fleet.BitLockerPINRequestFailed, "PIN rejected")

	// The agent collected the PIN too long ago and never reported, so Fleet stops waiting for it.
	unreported := newBitLockerPINHost(t, ds)
	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, unreported, "encrypted-pin"))
	_, unreportedUUID, err := ds.TakeBitLockerPINRequest(ctx, unreported)
	require.NoError(t, err)
	ageBitLockerPINRequest(t, ds, unreported.ID, "updated_at", fleet.BitLockerPINResultTimeout+time.Minute)
	// Refused as soon as it times out.
	require.True(t, fleet.IsNotFound(
		ds.SetBitLockerPINRequestOutcome(ctx, unreported, unreportedUUID, fleet.BitLockerPINRequestSet, "")))

	// Collected recently, so the agent may still report.
	recentlyCollected := newBitLockerPINHost(t, ds)
	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, recentlyCollected, "encrypted-pin"))
	_, _, err = ds.TakeBitLockerPINRequest(ctx, recentlyCollected)
	require.NoError(t, err)
	ageBitLockerPINRequest(t, ds, recentlyCollected.ID, "updated_at", fleet.BitLockerPINResultTimeout-time.Minute)

	// Expires in this same run, so it only just became terminal and must stay visible as a timeout for a full day.
	justExpired := newBitLockerPINHost(t, ds)
	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, justExpired, "encrypted-pin"))
	ageBitLockerPINRequest(t, ds, justExpired.ID, "created_at", fleet.BitLockerPINRequestTTL+time.Minute)

	// Still within its TTL, so the agent can still collect it and the flag must survive the run.
	stillPending := newBitLockerPINHost(t, ds)
	require.NoError(t, ds.QueueBitLockerPINRequest(ctx, stillPending, "encrypted-pin"))

	require.NoError(t, ds.CleanupExpiredBitLockerPINRequests(ctx))

	require.False(t, exists(t, oldSet.ID), "a set row past retention is reaped")
	require.False(t, exists(t, oldFailed.ID), "a failed row past retention is reaped")
	require.True(t, exists(t, recentFailed.ID), "a recently finished row is kept")

	for _, hostID := range []uint{justExpired.ID, unreported.ID} {
		req, err := ds.GetBitLockerPINRequest(ctx, hostID)
		require.NoError(t, err)
		require.Equal(t, fleet.BitLockerPINRequestFailed, req.Status)
		require.Equal(t, fleet.BitLockerPINRequestTimedOutError, req.Error)
		require.Nil(t, storedBitLockerPIN(t, ds, hostID), "a timed-out submission does not keep its ciphertext")
	}

	req, err := ds.GetBitLockerPINRequest(ctx, recentlyCollected.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.BitLockerPINRequestDelivered, req.Status)

	// Still refused once the cron has retired it, so orbit drops the outcome. osquery still shows whether the PIN was set.
	err = ds.SetBitLockerPINRequestOutcome(ctx, unreported, unreportedUUID, fleet.BitLockerPINRequestSet, "")
	require.True(t, fleet.IsNotFound(err))

	// The enrollment flag is what the config poll reads, so retiring a submission has to clear it.
	require.False(t, bitLockerPINPending(t, ds, justExpired.UUID), "expiring a submission clears the enrollment flag")
	require.True(t, bitLockerPINPending(t, ds, stillPending.UUID), "a collectable submission keeps the enrollment flag")
}
