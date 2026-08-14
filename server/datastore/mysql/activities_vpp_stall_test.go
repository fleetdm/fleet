package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestVPPInstallVerificationStall reproduces the Primo iOS report (Unthread
// #17867): an App Store (VPP) install that the device Acknowledges but that is
// never verified leaves the host's upcoming activity queue blocked forever,
// with no recovery path, while the automatic-update path keeps appending new
// installs for the same app behind it.
//
// The two subtests here document defects, not desired behaviour, and both are
// expected to FAIL until they are fixed:
//
//   - NoRecoveryFromUnverifiedHead: nothing rescues an acked-but-unverified
//     head-of-queue item. UnblockHostsUpcomingActivityQueue only considers
//     hosts with *no* activated row (activities.go:944-955), and the
//     vpp_verify_timeout is only evaluated inside the VERIFY-VPP-INSTALLS-
//     result handler (apple_mdm_cmd_results.go), so it never fires once that
//     chain breaks.
//   - NoDedupAgainstQueuedInstall: the duplicate guards read only
//     host_vpp_software_installs, whose rows are created at activation time,
//     so queued-but-unactivated installs are invisible and every pass appends
//     another row.
//
// The already-correct behaviour (a successful install waits for verification
// before activating the next item) is covered by testListHostUpcomingActivities
// and is deliberately not re-asserted here.
func TestVPPInstallVerificationStall(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NoRecoveryFromUnverifiedHead", testNoRecoveryFromUnverifiedHead},
		{"NoDedupAgainstQueuedInstall", testNoDedupAgainstQueuedInstall},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// stalledVPPHost enrolls an iOS host, inserts an iOS VPP app, enqueues an
// install for it, and drives that install to the acked-but-unverified state
// that blocks the queue. It returns the host, the app and the stalled install's
// execution ID.
func stalledVPPHost(t *testing.T, ds *Datastore, hostName, hostUUID string) (*fleet.Host, *fleet.VPPApp, string) {
	ctx := t.Context()
	activitySvc := NewTestActivityService(t, ds)
	test.CreateInsertGlobalVPPToken(t, ds)

	host := test.NewHost(t, ds, hostName, "10.10.10.1", hostUUID, hostUUID, time.Now(), test.WithPlatform("ios"))
	nanoEnrollAndSetHostMDMData(t, ds, host, false)

	app := &fleet.VPPApp{
		Name: "stalled_app_" + hostUUID,
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{AdamID: "adam-" + hostUUID, Platform: fleet.IOSPlatform},
		},
		BundleIdentifier: "bundle-" + hostUUID,
	}
	_, err := ds.InsertVPPAppWithTeam(ctx, app, nil)
	require.NoError(t, err)

	// Enqueue the install the way the automatic-update path does
	// (ForScheduledUpdates is what sets fleet_initiated, per
	// fleet.HostSoftwareInstallOptions.IsFleetInitiated).
	stalledExecID := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, host.ID, app.VPPAppID, stalledExecID, "evt-stalled",
		fleet.HostSoftwareInstallOptions{ForScheduledUpdates: true})
	require.NoError(t, err)

	// The device pulls the InstallApplication command.
	nanoCtx := &mdm.Request{EnrollID: &mdm.EnrollID{ID: host.UUID}, Context: ctx}
	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)
	cmd, err := nanoDB.RetrieveNextCommand(nanoCtx, false)
	require.NoError(t, err)
	require.Equal(t, stalledExecID, cmd.CommandUUID)
	require.Equal(t, "InstallApplication", cmd.Command.Command.RequestType)

	// The device installs it successfully and acknowledges. This is the
	// customer's state: the phone did its job and reports healthy.
	err = nanoDB.StoreCommandReport(nanoCtx, &mdm.CommandResults{
		CommandUUID: stalledExecID,
		Status:      "Acknowledged",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	})
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      host.ID,
		AppStoreID:  app.VPPAppTeam.AdamID,
		CommandUUID: stalledExecID,
		Status:      string(fleet.SoftwareInstalled),
	})
	require.NoError(t, err)

	// Verification never arrives, so the row stays activated and the queue is
	// blocked. Confirm that precondition before asserting anything about it.
	requireStillActivated(t, ds, stalledExecID)

	return host, app, stalledExecID
}

// requireStillActivated asserts the given upcoming activity is present and
// still holds the head of the queue (activated_at set, no verification).
func requireStillActivated(t *testing.T, ds *Datastore, execID string) {
	t.Helper()
	ctx := t.Context()

	var got []struct {
		ActivatedAtSet bool `db:"activated_at_set"`
		Verified       bool `db:"verified"`
		VerifyFailed   bool `db:"verify_failed"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &got, `
			SELECT
				ua.activated_at IS NOT NULL          AS activated_at_set,
				hvsi.verification_at IS NOT NULL     AS verified,
				hvsi.verification_failed_at IS NOT NULL AS verify_failed
			FROM upcoming_activities ua
			LEFT JOIN host_vpp_software_installs hvsi
				ON hvsi.command_uuid = ua.execution_id
			WHERE ua.execution_id = ?`, execID)
	})
	require.Len(t, got, 1, "stalled install should still be in upcoming_activities")
	require.True(t, got[0].ActivatedAtSet, "stalled install should still be activated (holding the queue)")
	require.False(t, got[0].Verified, "stalled install must not be verified for this repro")
	require.False(t, got[0].VerifyFailed, "stalled install must not be verify-failed for this repro")
}

// testNoRecoveryFromUnverifiedHead asserts that a host whose queue is blocked
// by an acked-but-unverified VPP install eventually recovers.
//
// It fails today: no reaper exists, and UnblockHostsUpcomingActivityQueue skips
// this host because it only targets hosts with no activated row.
func testNoRecoveryFromUnverifiedHead(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "recovery", "recovery@example.com", false)

	host, _, stalledExecID := stalledVPPHost(t, ds, "stall.local", "stall-uuid")

	// A script queued behind the stalled install. This is the blast radius the
	// customer would not connect to app installs: the queue is shared, so a
	// stuck VPP install blocks scripts too.
	script, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         host.ID,
		ScriptContents: "echo blocked",
		UserID:         &user.ID,
		SyncRequest:    true,
	})
	require.NoError(t, err)

	// Both are queued, the stalled install still at the head.
	checkUpcomingActivities(t, ds, host, stalledExecID, script.ExecutionID)

	// Age the whole queue well past any plausible verification timeout
	// (vpp_verify_timeout defaults to 10 minutes) so that a wall-clock reaper,
	// if one existed, would have fired. The customer's oldest row is ~72 days.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			UPDATE upcoming_activities
			SET created_at = DATE_SUB(NOW(6), INTERVAL 72 DAY),
			    activated_at = IF(activated_at IS NULL, NULL, DATE_SUB(NOW(6), INTERVAL 72 DAY))
			WHERE host_id = ?`, host.ID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			UPDATE host_vpp_software_installs
			SET created_at = DATE_SUB(NOW(6), INTERVAL 72 DAY)
			WHERE command_uuid = ?`, stalledExecID)
		return err
	})

	// The documented recovery mechanism for a blocked queue.
	n, err := ds.UnblockHostsUpcomingActivityQueue(ctx, 10)
	require.NoError(t, err)

	// EXPECTED (currently fails): the stalled install is resolved one way or
	// another and the script behind it runs. A host cannot be left permanently
	// unable to execute any activity because one install was never verified.
	require.Equal(t, 1, n, "expected the blocked host to be unblocked")
	checkUpcomingActivities(t, ds, host, script.ExecutionID)
}

// testNoDedupAgainstQueuedInstall asserts that the install path does not queue a
// second install for an app that already has one pending.
//
// It fails today: MapAdamIDsRecentInstalls and
// MapAdamIDsPendingInstallVerification both read only
// host_vpp_software_installs, whose rows are created at activation time, so
// queued-but-unactivated installs are invisible to the duplicate guards. This
// is the mechanism behind the unbounded growth (1,642 -> 1,816 rows on one
// host, with same-app duplicates seconds apart).
func testNoDedupAgainstQueuedInstall(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host, headApp, stalledExecID := stalledVPPHost(t, ds, "dedup.local", "dedup-uuid")

	// A second app, queued *behind* the stalled head. Because the head never
	// completes, this one is never activated, so it never gets a
	// host_vpp_software_installs row. This is Citymapper/Dashlane in the
	// customer's report.
	behindApp := &fleet.VPPApp{
		Name: "queued_behind_app",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{AdamID: "adam-behind", Platform: fleet.IOSPlatform},
		},
		BundleIdentifier: "bundle-behind",
	}
	_, err := ds.InsertVPPAppWithTeam(ctx, behindApp, nil)
	require.NoError(t, err)

	behindExecID := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, host.ID, behindApp.VPPAppID, behindExecID, "evt-behind",
		fleet.HostSoftwareInstallOptions{ForScheduledUpdates: true})
	require.NoError(t, err)

	// Confirm the shape of the repro: head activated, the one behind it not.
	requireStillActivated(t, ds, stalledExecID)
	var behindActivated bool
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &behindActivated,
			`SELECT activated_at IS NOT NULL FROM upcoming_activities WHERE execution_id = ?`, behindExecID)
	})
	require.False(t, behindActivated, "install behind the stalled head must not be activated")

	// The two guards the automatic-update path consults before queueing.
	recent, err := ds.MapAdamIDsRecentInstalls(ctx, host.ID, 3600)
	require.NoError(t, err)
	pendingVerification, err := ds.MapAdamIDsPendingInstallVerification(ctx, host.ID)
	require.NoError(t, err)

	seenByEitherGuard := func(adamID string) bool {
		_, isRecent := recent[adamID]
		_, isPending := pendingVerification[adamID]
		return isRecent || isPending
	}

	// Control: the activated head IS visible. MapAdamIDsPendingInstallVerification
	// explicitly covers "acknowledged but awaiting verification", so the guards
	// are not uniformly broken — which is why the head's own app does not get
	// duplicated.
	require.True(t, seenByEitherGuard(headApp.VPPAppTeam.AdamID),
		"the activated head should be visible to the duplicate guards")

	// EXPECTED (currently fails): the install queued behind the head is also
	// visible, so the next automatic-update pass skips that app instead of
	// appending another row for it. Both guards read only
	// host_vpp_software_installs, which is written at activation time, so today
	// this install is invisible and every pass stacks another copy.
	require.True(t, seenByEitherGuard(behindApp.VPPAppTeam.AdamID),
		"a queued but not-yet-activated install for adam_id %s should be visible to the duplicate guards",
		behindApp.VPPAppTeam.AdamID)
}
