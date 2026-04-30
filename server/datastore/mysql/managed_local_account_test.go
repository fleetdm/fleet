package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedLocalAccount(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SaveAndGetPassword", testManagedLocalAccountSaveAndGetPassword},
		{"GetStatus", testManagedLocalAccountGetStatus},
		{"SetStatus", testManagedLocalAccountSetStatus},
		{"GetByCommandUUID", testManagedLocalAccountGetByCommandUUID},
		{"UpsertOverwrites", testManagedLocalAccountUpsertOverwrites},
		{"NotFound", testManagedLocalAccountNotFound},
		{"GetSetAccountUUID", testManagedLocalAccountGetSetAccountUUID},
		{"MarkViewed", testManagedLocalAccountMarkViewed},
		{"InitiateRotation", testManagedLocalAccountInitiateRotation},
		{"CompleteRotation", testManagedLocalAccountCompleteRotation},
		{"FailRotation", testManagedLocalAccountFailRotation},
		{"ClearRotation", testManagedLocalAccountClearRotation},
		{"DeferredRotation", testManagedLocalAccountDeferredRotation},
		{"GetForAutoRotation", testManagedLocalAccountGetForAutoRotation},
		{"GetByPendingCommandUUID", testManagedLocalAccountGetByPendingCommandUUID},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testManagedLocalAccountSaveAndGetPassword(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-1"
	password := "TEST-PASS-WORD1"
	cmdUUID := "cmd-uuid-1"

	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, password, cmdUUID)
	require.NoError(t, err)

	got, err := ds.GetHostManagedLocalAccountPassword(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, "_fleetadmin", got.Username)
	assert.Equal(t, password, got.Password)
	assert.False(t, got.UpdatedAt.IsZero())
}

func testManagedLocalAccountGetStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-status"
	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, "pass", "cmd-status")
	require.NoError(t, err)

	// Initially status is NULL in DB → should return "pending".
	// New (post-rotation) semantics: password is available whenever encrypted_password
	// is set and status != 'failed'. The pre-ack NULL status with a stored password
	// is therefore "available" too.
	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, "pending", *status.Status)
	assert.True(t, status.PasswordAvailable)
	assert.False(t, status.PendingRotation)
	assert.Nil(t, status.AutoRotateAt)

	// After setting to verified, password should be available.
	err = ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryVerified)
	require.NoError(t, err)

	status, err = ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryVerified), *status.Status)
	assert.True(t, status.PasswordAvailable)
}

func testManagedLocalAccountSetStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-set-status"
	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, "pass", "cmd-set-status")
	require.NoError(t, err)

	// Set to failed.
	err = ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryFailed)
	require.NoError(t, err)

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryFailed), *status.Status)
	assert.False(t, status.PasswordAvailable)
}

func testManagedLocalAccountGetByCommandUUID(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a real host so the host lookup in GetManagedLocalAccountByCommandUUID succeeds.
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "managed-account-host",
		OsqueryHostID:   new("managed-account-osquery-1"),
		NodeKey:         new("managed-account-node-1"),
		UUID:            "host-uuid-cmd",
		Platform:        "darwin",
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		PolicyUpdatedAt: ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
	})
	require.NoError(t, err)

	cmdUUID := "cmd-uuid-lookup"
	err = ds.SaveHostManagedLocalAccount(ctx, host.UUID, "pass", cmdUUID)
	require.NoError(t, err)

	got, err := ds.GetManagedLocalAccountByCommandUUID(ctx, cmdUUID)
	require.NoError(t, err)
	assert.Equal(t, host.UUID, got.UUID)
	assert.Equal(t, host.ID, got.ID)
}

func testManagedLocalAccountUpsertOverwrites(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a real host so the host lookup succeeds.
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "managed-account-upsert-host",
		OsqueryHostID:   new("managed-account-osquery-2"),
		NodeKey:         new("managed-account-node-2"),
		UUID:            "host-uuid-upsert",
		Platform:        "darwin",
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		PolicyUpdatedAt: ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
	})
	require.NoError(t, err)

	// First save.
	err = ds.SaveHostManagedLocalAccount(ctx, host.UUID, "old-pass", "cmd-old")
	require.NoError(t, err)
	err = ds.SetHostManagedLocalAccountStatus(ctx, host.UUID, fleet.MDMDeliveryVerified)
	require.NoError(t, err)

	// Upsert with new password and command UUID should reset status to NULL (pending).
	err = ds.SaveHostManagedLocalAccount(ctx, host.UUID, "new-pass", "cmd-new")
	require.NoError(t, err)

	got, err := ds.GetHostManagedLocalAccountPassword(ctx, host.UUID)
	require.NoError(t, err)
	assert.Equal(t, "new-pass", got.Password)

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, "pending", *status.Status)

	// Command UUID should be the new one.
	foundHost, err := ds.GetManagedLocalAccountByCommandUUID(ctx, "cmd-new")
	require.NoError(t, err)
	assert.Equal(t, host.UUID, foundHost.UUID)

	// Old command UUID should no longer match.
	_, err = ds.GetManagedLocalAccountByCommandUUID(ctx, "cmd-old")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testManagedLocalAccountNotFound(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	_, err := ds.GetHostManagedLocalAccountPassword(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetHostManagedLocalAccountStatus(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetManagedLocalAccountByCommandUUID(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetManagedLocalAccountUUID(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testManagedLocalAccountGetSetAccountUUID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-account-uuid"
	accountUUID := "AAAAAAAA-BBBB-CCCC-DDDD-000000000001"

	// No row yet.
	_, err := ds.GetManagedLocalAccountUUID(ctx, hostUUID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Set before row exists is a no-op (no error). Get still returns NotFound.
	require.NoError(t, ds.SetManagedLocalAccountUUID(ctx, hostUUID, accountUUID))
	_, err = ds.GetManagedLocalAccountUUID(ctx, hostUUID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Create the row (account_uuid NULL by default).
	require.NoError(t, ds.SaveHostManagedLocalAccount(ctx, hostUUID, "pw", "cmd-1"))

	got, err := ds.GetManagedLocalAccountUUID(ctx, hostUUID)
	require.NoError(t, err)
	assert.Nil(t, got)

	// First Set populates account_uuid.
	require.NoError(t, ds.SetManagedLocalAccountUUID(ctx, hostUUID, accountUUID))
	got, err = ds.GetManagedLocalAccountUUID(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, accountUUID, *got)

	// Second Set with a different value updates it.
	otherUUID := "AAAAAAAA-BBBB-CCCC-DDDD-000000000002"
	require.NoError(t, ds.SetManagedLocalAccountUUID(ctx, hostUUID, otherUUID))
	got, err = ds.GetManagedLocalAccountUUID(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, otherUUID, *got)
}

// newManagedLocalAccountTestHost spins up a host row + managed-local-account row in a
// state ready for rotation tests: encrypted_password set, account_uuid captured,
// status='verified'. Returns the host UUID.
func newManagedLocalAccountTestHost(t *testing.T, ds *Datastore, suffix string) string {
	t.Helper()
	ctx := t.Context()
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "rot-host-" + suffix,
		ComputerName:    "Rot Host " + suffix,
		OsqueryHostID:   new("rot-osq-" + suffix),
		NodeKey:         new("rot-node-" + suffix),
		UUID:            "rot-host-uuid-" + suffix,
		Platform:        "darwin",
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		PolicyUpdatedAt: ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
	})
	require.NoError(t, err)
	require.NoError(t, ds.SaveHostManagedLocalAccount(ctx, host.UUID, "init-pass-"+suffix, "init-cmd-"+suffix))
	require.NoError(t, ds.SetManagedLocalAccountUUID(ctx, host.UUID, "AAAAAAAA-BBBB-CCCC-DDDD-"+suffix+"00"))
	require.NoError(t, ds.SetHostManagedLocalAccountStatus(ctx, host.UUID, fleet.MDMDeliveryVerified))
	return host.UUID
}

func testManagedLocalAccountMarkViewed(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "view0001")

	// First view sets auto_rotate_at ~65 minutes in the future and flips status to pending.
	rotateAt, err := ds.MarkManagedLocalAccountPasswordViewed(ctx, hostUUID)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(65*time.Minute), rotateAt, 30*time.Second)

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryPending), *status.Status)
	require.NotNil(t, status.AutoRotateAt)
	assert.True(t, status.PasswordAvailable)
	assert.False(t, status.PendingRotation)

	// Second view inside the window does NOT extend the timer — the same auto_rotate_at
	// is returned. We allow 1s of drift for the read clock.
	rotateAt2, err := ds.MarkManagedLocalAccountPasswordViewed(ctx, hostUUID)
	require.NoError(t, err)
	assert.WithinDuration(t, rotateAt, rotateAt2, time.Second)

	// Mark failed → next view returns notFound (failed rows are ineligible).
	require.NoError(t, ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryFailed))
	_, err = ds.MarkManagedLocalAccountPasswordViewed(ctx, hostUUID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testManagedLocalAccountInitiateRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "init0002")

	// Happy path
	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "new-pending-1", "rot-cmd-1"))

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	assert.True(t, status.PendingRotation)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryPending), *status.Status)

	// Second initiate while one is pending → typed error.
	err = ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "nope", "rot-cmd-2")
	require.ErrorIs(t, err, fleet.ErrManagedLocalAccountRotationPending)

	// Clear pending so we can probe the not-eligible path. Then mark failed.
	require.NoError(t, ds.ClearManagedLocalAccountRotation(ctx, hostUUID))
	require.NoError(t, ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryFailed))
	err = ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "nope", "rot-cmd-3")
	require.ErrorIs(t, err, fleet.ErrManagedLocalAccountNotEligible)

	// Missing row → notFound.
	err = ds.InitiateManagedLocalAccountRotation(ctx, "no-such-host", "nope", "rot-cmd-x")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testManagedLocalAccountCompleteRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "comp0003")

	// Mark viewed so auto_rotate_at is set; this also exercises the clear-on-complete path.
	_, err := ds.MarkManagedLocalAccountPasswordViewed(ctx, hostUUID)
	require.NoError(t, err)

	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "the-new-password", "rot-cmd-comp"))
	require.NoError(t, ds.CompleteManagedLocalAccountRotation(ctx, hostUUID, "rot-cmd-comp"))

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryVerified), *status.Status)
	assert.False(t, status.PendingRotation)
	assert.Nil(t, status.AutoRotateAt)

	pwd, err := ds.GetHostManagedLocalAccountPassword(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, "the-new-password", pwd.Password)

	// Mismatched cmdUUID → notFound.
	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "another-password", "rot-cmd-comp2"))
	err = ds.CompleteManagedLocalAccountRotation(ctx, hostUUID, "wrong-cmd")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testManagedLocalAccountFailRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "fail0004")

	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "rotation-password", "rot-cmd-fail"))
	require.NoError(t, ds.FailManagedLocalAccountRotation(ctx, hostUUID, "rot-cmd-fail", "device returned error"))

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryFailed), *status.Status)
	assert.False(t, status.PendingRotation)
	assert.False(t, status.PasswordAvailable, "failed rotations hide the password until reset")
	assert.Nil(t, status.AutoRotateAt)

	// Old (still-good) password is preserved.
	pwd, err := ds.GetHostManagedLocalAccountPassword(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, "init-pass-fail0004", pwd.Password)
}

func testManagedLocalAccountClearRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "clr00005")

	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "rotation-password", "rot-cmd-clr"))

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	assert.True(t, status.PendingRotation)

	require.NoError(t, ds.ClearManagedLocalAccountRotation(ctx, hostUUID))

	status, err = ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	assert.False(t, status.PendingRotation)

	// Idempotent.
	require.NoError(t, ds.ClearManagedLocalAccountRotation(ctx, hostUUID))
}

func testManagedLocalAccountDeferredRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "def00006")

	// Defer: status='pending', auto_rotate_at=NOW(6), initiated_by_fleet=0.
	require.NoError(t, ds.MarkManagedLocalAccountRotationDeferred(ctx, hostUUID))

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryPending), *status.Status)
	require.NotNil(t, status.AutoRotateAt)

	// Idempotent (no error on second call).
	require.NoError(t, ds.MarkManagedLocalAccountRotationDeferred(ctx, hostUUID))
}

func testManagedLocalAccountGetForAutoRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Eligible: viewed (auto_rotate_at in past), password set, account_uuid set.
	dueHost := newManagedLocalAccountTestHost(t, ds, "due00007")
	_, err := ds.MarkManagedLocalAccountPasswordViewed(ctx, dueHost)
	require.NoError(t, err)
	// Backdate auto_rotate_at into the past.
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_managed_local_account_passwords SET auto_rotate_at = NOW(6) - INTERVAL 1 MINUTE WHERE host_uuid = ?`, dueHost)
	require.NoError(t, err)

	// Ineligible: not viewed (auto_rotate_at NULL).
	notViewed := newManagedLocalAccountTestHost(t, ds, "noview08")

	// Ineligible: viewed but in the future.
	future := newManagedLocalAccountTestHost(t, ds, "fut00009")
	_, err = ds.MarkManagedLocalAccountPasswordViewed(ctx, future)
	require.NoError(t, err)

	// Ineligible: pending rotation already.
	pending := newManagedLocalAccountTestHost(t, ds, "pen00010")
	_, err = ds.MarkManagedLocalAccountPasswordViewed(ctx, pending)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_managed_local_account_passwords SET auto_rotate_at = NOW(6) - INTERVAL 1 MINUTE WHERE host_uuid = ?`, pending)
	require.NoError(t, err)
	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, pending, "p", "p-cmd"))

	// Ineligible: failed.
	failed := newManagedLocalAccountTestHost(t, ds, "fai00011")
	require.NoError(t, ds.SetHostManagedLocalAccountStatus(ctx, failed, fleet.MDMDeliveryFailed))
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_managed_local_account_passwords SET auto_rotate_at = NOW(6) - INTERVAL 1 MINUTE WHERE host_uuid = ?`, failed)
	require.NoError(t, err)

	// Ineligible: no account_uuid.
	noUUID := newManagedLocalAccountTestHost(t, ds, "nou00012")
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_managed_local_account_passwords SET account_uuid = NULL, auto_rotate_at = NOW(6) - INTERVAL 1 MINUTE WHERE host_uuid = ?`, noUUID)
	require.NoError(t, err)

	// Ineligible: deferred-but-no-uuid path: even with auto_rotate_at in the past, missing
	// account_uuid filters the row out (cron will pick up once UUID lands).
	rows, err := ds.GetManagedLocalAccountsForAutoRotation(ctx)
	require.NoError(t, err)

	got := make(map[string]struct{})
	for _, r := range rows {
		got[r.HostUUID] = struct{}{}
	}
	_, hasDue := got[dueHost]
	_, hasNotViewed := got[notViewed]
	_, hasFuture := got[future]
	_, hasPending := got[pending]
	_, hasFailed := got[failed]
	_, hasNoUUID := got[noUUID]
	assert.True(t, hasDue, "due host should be returned")
	assert.False(t, hasNotViewed, "non-viewed host should not be returned")
	assert.False(t, hasFuture, "future-rotation host should not be returned")
	assert.False(t, hasPending, "host with pending rotation should not be returned")
	assert.False(t, hasFailed, "failed host should not be returned")
	assert.False(t, hasNoUUID, "host without account_uuid should not be returned")

	// Confirm we surface initiated_by_fleet=true for view-driven rows.
	for _, r := range rows {
		if r.HostUUID == dueHost {
			assert.True(t, r.InitiatedByFleet)
			assert.NotEmpty(t, r.AccountUUID)
		}
	}
}

func testManagedLocalAccountGetByPendingCommandUUID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := newManagedLocalAccountTestHost(t, ds, "pen00013")
	require.NoError(t, ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, "p", "rot-cmd-pend"))

	got, err := ds.GetManagedLocalAccountByPendingCommandUUID(ctx, "rot-cmd-pend")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, hostUUID, got.UUID)

	// Wrong UUID → notFound.
	_, err = ds.GetManagedLocalAccountByPendingCommandUUID(ctx, "no-such-cmd")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}
