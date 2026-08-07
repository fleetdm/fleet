package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	nanomdm_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestRepro50680 reproduces the dedup blindness described in
// https://github.com/fleetdm/fleet/issues/50680: the two guards used by
// handleScheduledUpdates read only host_vpp_software_installs, whose rows are
// created at activation time, so an install still waiting in
// upcoming_activities is invisible to both.
func TestRepro50680(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "repro50680"})
	require.NoError(t, err)

	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Org"+t.Name(), "Loc"+t.Name())
	require.NoError(t, err)
	tok, err := ds.InsertVPPToken(ctx, dataToken)
	require.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tok.ID, []uint{tm.ID})
	require.NoError(t, err)

	user := test.NewUser(t, ds, "Alice", "alice50680@example.com", true)

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "ios-50680",
		UUID:           uuid.NewString(),
		Platform:       string(fleet.IOSPlatform),
		HardwareSerial: uuid.NewString(),
		TeamID:         &tm.ID,
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	mkApp := func(adamID, bundle string) string {
		app := &fleet.VPPApp{
			VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: adamID, Platform: fleet.IOSPlatform}},
			Name:             adamID,
			BundleIdentifier: bundle,
			LatestVersion:    "2.0.0",
		}
		va, err := ds.InsertVPPAppWithTeam(ctx, app, &tm.ID)
		require.NoError(t, err)
		return va.AdamID
	}

	adamBlocker := mkApp("adam_blocker", "com.app.blocker")
	adamDup := mkApp("adam_dup", "com.app.dup")

	// 1. Queue + activate the blocker; it activates immediately (empty queue),
	//    which is what creates its host_vpp_software_installs row.
	blockerCmd := createVPPAppInstallRequest(t, ds, host, adamBlocker, user)

	var activatedCount int
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &activatedCount,
		`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activated_at IS NOT NULL`, host.ID))
	require.Equal(t, 1, activatedCount, "blocker should be activated")

	var hvsiCount int
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &hvsiCount,
		`SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ?`, host.ID))
	require.Equal(t, 1, hvsiCount, "activation creates the install row")

	// 2. Device acknowledges the install but verification never completes.
	//    No activity is recorded, so the upcoming activity stays activated.
	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)
	require.NoError(t, nanoDB.StoreCommandReport(
		&mdm.Request{EnrollID: &mdm.EnrollID{ID: host.UUID}, Context: ctx},
		&mdm.CommandResults{CommandUUID: blockerCmd, Status: "Acknowledged", Raw: []byte(`<?xml version="1.0" encoding="UTF-8"?>`)},
	))

	// 3. Queue an install for a different app. It cannot activate (head of
	//    queue is already activated), so no host_vpp_software_installs row.
	_ = createVPPAppInstallRequest(t, ds, host, adamDup, user)

	require.NoError(t, ds.writer(ctx).GetContext(ctx, &hvsiCount,
		`SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ?`, host.ID))
	require.Equal(t, 1, hvsiCount, "queued-but-unactivated install has no install row")

	var queued int
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &queued,
		`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activated_at IS NULL`, host.ID))
	require.Equal(t, 1, queued, "the dup app is waiting in the queue")

	// 4. Both guards used by handleScheduledUpdates are blind to it.
	pending, err := ds.MapAdamIDsPendingInstallVerification(ctx, host.ID)
	require.NoError(t, err)
	t.Logf("MapAdamIDsPendingInstallVerification -> %v", pending)
	require.Contains(t, pending, adamBlocker, "the activated head IS seen")
	require.NotContains(t, pending, adamDup,
		"BUG: a queued install for adam_dup is invisible to the pending-verification guard")

	recent, err := ds.MapAdamIDsRecentInstalls(ctx, host.ID, 3600)
	require.NoError(t, err)
	t.Logf("MapAdamIDsRecentInstalls -> %v", recent)
	require.NotContains(t, recent, adamDup,
		"BUG: a queued install for adam_dup is invisible to the recent-install guard")

	// The policy path guard has the same blind spot.
	pendingPolicy, err := ds.MapAdamIDsPendingInstall(ctx, host.ID)
	require.NoError(t, err)
	t.Logf("MapAdamIDsPendingInstall -> %v", pendingPolicy)
	require.NotContains(t, pendingPolicy, adamDup,
		"BUG: policy-path guard is blind to queued installs too")

	// 5. Consequence: nothing stops the next pass from queueing the same app again.
	for i := 0; i < 5; i++ {
		_ = createVPPAppInstallRequest(t, ds, host, adamDup, user)
	}
	var dupRows int
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &dupRows, `
		SELECT COUNT(*) FROM upcoming_activities ua
		JOIN vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
		WHERE ua.host_id = ? AND vaua.adam_id = ?`, host.ID, adamDup))
	t.Logf("queued copies of %s: %d", adamDup, dupRows)
	require.Equal(t, 6, dupRows, "unbounded duplicate queueing")
}
