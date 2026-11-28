package mysql

import (
	"slices"
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

// TestVPPSplitGaps pins the public behavior of App Store and in-house app installs through
// datastore methods that exist on main, so it can run unchanged before and after the install
// tables are restructured. It never reads or writes the new tables, except for the host delete
// check that asserts they are cleaned up when present.
func TestVPPSplitGaps(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"RetryHistory", testVPPSplitRetryHistory},
		{"Removed", testVPPSplitRemoved},
		{"NeedsResend", testVPPSplitNeedsResend},
		{"AndroidPending", testVPPSplitAndroidPending},
		{"AppleVerificationTimeout", testVPPSplitAppleVerificationTimeout},
		{"AndroidVerificationTimeout", testVPPSplitAndroidVerificationTimeout},
		{"CancelActivatedInstall", testVPPSplitCancelActivated},
		{"HostDelete", testVPPSplitHostDelete},
		{"FleetTransfer", testVPPSplitFleetTransfer},
		{"TokenRemoval", testVPPSplitTokenRemoval},
		{"AppDeletionCascade", testVPPSplitAppDeletionCascade},
		{"NanoOrphanGuard", testVPPSplitNanoOrphanGuard},
		{"FailedInstallWindow", testVPPSplitFailedInstallWindow},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func vppSplitHost(t *testing.T, ds *Datastore, name, platform string, teamID *uint) *fleet.Host {
	host, err := ds.NewHost(t.Context(), &fleet.Host{
		Hostname:       name,
		UUID:           uuid.NewString(),
		OsqueryHostID:  new("osq-" + name),
		NodeKey:        new("key-" + name),
		Platform:       platform,
		HardwareSerial: uuid.NewString(),
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)
	if teamID != nil {
		require.NoError(t, ds.AddHostsToTeam(t.Context(), fleet.NewAddHostsToTeamParams(teamID, []uint{host.ID})))
		host.TeamID = teamID
	}
	return host
}

func vppSplitApp(t *testing.T, ds *Datastore, adamID string, platform fleet.InstallableDevicePlatform, teamID *uint) *fleet.VPPApp {
	app, err := ds.InsertVPPAppWithTeam(t.Context(), &fleet.VPPApp{
		Name:             "app " + adamID,
		BundleIdentifier: "com.split." + adamID,
		AdamID:           adamID,
		Platform:         platform,
	}, teamID, nil)
	require.NoError(t, err)
	return app
}

func vppSplitQueue(t *testing.T, ds *Datastore, host *fleet.Host, app *fleet.VPPApp, opts fleet.HostSoftwareInstallOptions) string {
	cmd := uuid.NewString()
	opts.VPPAppTeamID = app.AppTeamID
	require.NoError(t, ds.InsertHostVPPSoftwareInstall(t.Context(), host.ID, app.VPPAppID, cmd, uuid.NewString(), opts))
	return cmd
}

// vppSplitReport stores a command result without touching the install, like the MDM result
// handler does before it decides whether to retry.
func vppSplitReport(t *testing.T, ds *Datastore, host *fleet.Host, cmd, status string) {
	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)
	require.NoError(t, nanoDB.StoreCommandReport(&mdm.Request{EnrollID: &mdm.EnrollID{ID: host.UUID}, Context: t.Context()},
		&mdm.CommandResults{CommandUUID: cmd, Status: status, Raw: []byte(`<?xml version="1.0" encoding="UTF-8"?>`)}))
}

func vppSplitVerify(t *testing.T, ds *Datastore, host *fleet.Host, cmd string) {
	createVPPAppInstallResult(t, ds, host, cmd, fleet.MDMAppleStatusAcknowledged)
	require.NoError(t, ds.SetVPPInstallAsVerified(t.Context(), host.ID, cmd, uuid.NewString()))
}

// vppSplitRetry retries the install currently holding cmd and returns the new command uuid.
func vppSplitRetry(t *testing.T, ds *Datastore, cmd string) string {
	install, err := ds.GetHostVPPInstallByCommandUUID(t.Context(), cmd)
	require.NoError(t, err)
	require.NotNil(t, install)
	require.NoError(t, ds.RetryVPPInstall(t.Context(), install))

	var next string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &next,
			`SELECT command_uuid FROM host_vpp_software_installs WHERE host_id = ? AND retry_count = ? ORDER BY id DESC LIMIT 1`,
			install.HostID, install.RetryCount+1)
	})
	require.NotEqual(t, cmd, next)
	return next
}

// vppSplitStatus returns the status of the app in the host's software list, empty when the app is
// listed without an install or isn't listed at all.
func vppSplitStatus(t *testing.T, ds *Datastore, host *fleet.Host, adamID string) (status fleet.SoftwareInstallerStatus, listed bool) {
	titles, _, err := ds.ListHostSoftware(t.Context(), host, fleet.HostSoftwareTitleListOptions{IncludeAvailableForInstall: true})
	require.NoError(t, err)
	for _, title := range titles {
		if title.AppStoreApp != nil && title.AppStoreApp.AppStoreID == adamID {
			if title.Status == nil {
				return "", true
			}
			return *title.Status, true
		}
	}
	return "", false
}

func vppSplitSummary(t *testing.T, ds *Datastore, app *fleet.VPPApp) fleet.VPPAppStatusSummary {
	summaries, err := ds.GetSummaryHostVPPAppInstalls(t.Context(), []*fleet.VPPAppStoreApp{{VPPAppsTeamsID: app.AppTeamID}})
	require.NoError(t, err)
	require.NotNil(t, summaries[app.AppTeamID])
	return *summaries[app.AppTeamID]
}

// vppSplitHostsByStatus lists hosts by install status, resolving the app in the given fleet (none for no fleet).
func vppSplitHostsByStatus(t *testing.T, ds *Datastore, app *fleet.VPPApp, status fleet.SoftwareInstallerStatus, fleetID ...uint) []uint {
	opts := fleet.HostListOptions{SoftwareTitleIDFilter: &app.TitleID, SoftwareStatusFilter: &status}
	if len(fleetID) > 0 {
		opts.TeamFilter = &fleetID[0]
	}
	hosts, err := ds.ListHosts(t.Context(), fleet.TeamFilter{User: test.UserAdmin}, opts)
	require.NoError(t, err)
	ids := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		ids = append(ids, h.ID)
	}
	slices.Sort(ids)
	return ids
}

func vppSplitCount(t *testing.T, ds *Datastore, query string, args ...any) int {
	var n int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &n, query, args...)
	})
	return n
}

func vppSplitInHouseApp(t *testing.T, ds *Datastore, user *fleet.User, bundleID string) (appID, titleID uint) {
	appID, titleID, err := ds.MatchOrCreateSoftwareInstaller(t.Context(), &fleet.UploadSoftwareInstallerPayload{
		UserID:           user.ID,
		BundleIdentifier: bundleID,
		Filename:         bundleID + ".ipa",
		StorageID:        "storage-" + bundleID,
		Extension:        "ipa",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	return appID, titleID
}

func testVPPSplitRetryHistory(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "split_retry", fleet.MacOSPlatform, nil)
	recovered := vppSplitHost(t, ds, "recovered", "darwin", nil)
	exhausted := vppSplitHost(t, ds, "exhausted", "darwin", nil)

	// Each retry keeps the install in flight: the old command uuid stops resolving, the retry count
	// grows, and the host stays pending.
	cmdRecovered := vppSplitQueue(t, ds, recovered, app, fleet.HostSoftwareInstallOptions{})
	cmdExhausted := vppSplitQueue(t, ds, exhausted, app, fleet.HostSoftwareInstallOptions{})
	var oldCmds []string
	for i := 1; i <= 2; i++ {
		for host, cmd := range map[*fleet.Host]*string{recovered: &cmdRecovered, exhausted: &cmdExhausted} {
			vppSplitReport(t, ds, host, *cmd, fleet.MDMAppleStatusError)
			oldCmds = append(oldCmds, *cmd)
			*cmd = vppSplitRetry(t, ds, *cmd)

			install, err := ds.GetHostVPPInstallByCommandUUID(ctx, *cmd)
			require.NoError(t, err)
			require.NotNil(t, install)
			require.Equal(t, i, install.RetryCount)
			require.Equal(t, host.ID, install.HostID)

			status, _ := vppSplitStatus(t, ds, host, app.AdamID)
			require.Equal(t, fleet.SoftwareInstallPending, status)
		}
		require.Equal(t, fleet.VPPAppStatusSummary{Pending: 2}, vppSplitSummary(t, ds, app))
		require.ElementsMatch(t, []uint{recovered.ID, exhausted.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstallPending))
	}
	for _, old := range oldCmds {
		install, err := ds.GetHostVPPInstallByCommandUUID(ctx, old)
		require.NoError(t, err)
		require.Nil(t, install, old)
	}

	// One row per host: retries never add install rows.
	require.Equal(t, 2, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE adam_id = ?`, app.AdamID))

	// The recovered host verifies on the last attempt, the other one runs out of attempts.
	vppSplitVerify(t, ds, recovered, cmdRecovered)
	createVPPAppInstallResult(t, ds, exhausted, cmdExhausted, fleet.MDMAppleStatusError)

	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1, Failed: 1}, vppSplitSummary(t, ds, app))
	status, _ := vppSplitStatus(t, ds, recovered, app.AdamID)
	require.Equal(t, fleet.SoftwareInstalled, status)
	status, _ = vppSplitStatus(t, ds, exhausted, app.AdamID)
	require.Equal(t, fleet.SoftwareInstallFailed, status)
	require.Equal(t, []uint{recovered.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstalled))
	require.Equal(t, []uint{exhausted.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstallFailed))

	// The retry count of the final attempt survives, and the past activity data resolves for the
	// final command only.
	install, err := ds.GetHostVPPInstallByCommandUUID(ctx, cmdRecovered)
	require.NoError(t, err)
	require.NotNil(t, install)
	require.Equal(t, 2, install.RetryCount)
	_, act, err := ds.GetPastActivityDataForVPPAppInstall(ctx, &mdm.CommandResults{CommandUUID: cmdRecovered})
	require.NoError(t, err)
	require.NotNil(t, act)
	require.Equal(t, cmdRecovered, act.CommandUUID)
	require.NotEmpty(t, oldCmds)
	_, _, err = ds.GetPastActivityDataForVPPAppInstall(ctx, &mdm.CommandResults{CommandUUID: oldCmds[0]})
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
}

func testVPPSplitRemoved(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "split_removed", fleet.MacOSPlatform, nil)
	host := vppSplitHost(t, ds, "removed-host", "darwin", nil)
	other := vppSplitHost(t, ds, "removed-other", "darwin", nil)

	inventory := []fleet.Software{{Name: app.Name, Version: "1.0", Source: "apps", BundleIdentifier: app.BundleIdentifier}}
	for _, h := range []*fleet.Host{host, other} {
		_, err := ds.UpdateHostSoftware(ctx, h.ID, inventory)
		require.NoError(t, err)
		vppSplitVerify(t, ds, h, vppSplitQueue(t, ds, h, app, fleet.HostSoftwareInstallOptions{}))
	}
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 2}, vppSplitSummary(t, ds, app))

	// The inventory sync drops the app from one host only.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{})
	require.NoError(t, err)

	status, _ := vppSplitStatus(t, ds, host, app.AdamID)
	require.Empty(t, status) // listed or not, the removed install has no status
	status, _ = vppSplitStatus(t, ds, other, app.AdamID)
	require.Equal(t, fleet.SoftwareInstalled, status)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, app))
	// Hosts by status ignores removed on purpose.
	require.ElementsMatch(t, []uint{host.ID, other.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstalled))

	// A reinstall is a fresh current install.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, inventory)
	require.NoError(t, err)
	cmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{})
	status, _ = vppSplitStatus(t, ds, host, app.AdamID)
	require.Equal(t, fleet.SoftwareInstallPending, status)
	vppSplitVerify(t, ds, host, cmd)
	status, _ = vppSplitStatus(t, ds, host, app.AdamID)
	require.Equal(t, fleet.SoftwareInstalled, status)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 2}, vppSplitSummary(t, ds, app))
	install, err := ds.GetHostVPPInstallByCommandUUID(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, install)
	require.Zero(t, install.RetryCount)
}

func testVPPSplitNeedsResend(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "split_resend", fleet.IOSPlatform, nil)
	host := vppSplitHost(t, ds, "resend-host", "ios", nil)
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{{Name: app.Name, Version: "1.0", Source: "ios_apps", BundleIdentifier: app.BundleIdentifier}})
	require.NoError(t, err)

	platforms := []fleet.InstallableDevicePlatform{fleet.IOSPlatform, fleet.IPadOSPlatform}
	needsResend := func() bool {
		byTitle, err := ds.ListHostAppStoreAppInstallVersions(ctx, 0, []uint{host.ID}, platforms)
		require.NoError(t, err)
		install, ok := byTitle[app.TitleID][host.ID]
		require.True(t, ok)
		return install.NeedsResend
	}

	// An install that failed isn't considered, so only the verified one is listed.
	failedCmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{})
	createVPPAppInstallResult(t, ds, host, failedCmd, fleet.MDMAppleStatusError)
	byTitle, err := ds.ListHostAppStoreAppInstallVersions(ctx, 0, []uint{host.ID}, platforms)
	require.NoError(t, err)
	require.Empty(t, byTitle)

	cmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{})
	vppSplitVerify(t, ds, host, cmd)
	require.False(t, needsResend())

	// The SCIM path flags every row of the host and app through raw SQL on the request table.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_vpp_software_installs SET needs_resend = 1 WHERE host_id = ? AND adam_id = ? AND canceled = 0`, host.ID, app.AdamID)
		return err
	})
	require.True(t, needsResend())

	// The resend install is the new latest install and carries no flag.
	resendCmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{ForConfigurationResend: true})
	fromAutoUpdate, fromResend, err := ds.GetVPPInstallAutomationReasons(ctx, resendCmd)
	require.NoError(t, err)
	require.False(t, fromAutoUpdate)
	require.True(t, fromResend)
	vppSplitVerify(t, ds, host, resendCmd)
	require.False(t, needsResend())
}

func testVPPSplitAndroidPending(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "com.split.android", fleet.AndroidPlatform, nil)
	failedHost, err := ds.NewAndroidHost(ctx, createAndroidHost(uuid.NewString()), false)
	require.NoError(t, err)
	installedHost, err := ds.NewAndroidHost(ctx, createAndroidHost(uuid.NewString()), false)
	require.NoError(t, err)
	pendingHost, err := ds.NewAndroidHost(ctx, createAndroidHost(uuid.NewString()), false)
	require.NoError(t, err)

	cmds := map[uint]string{}
	for _, h := range []*fleet.AndroidHost{failedHost, installedHost, pendingHost} {
		cmds[h.Host.ID] = uuid.NewString()
		require.NoError(t, ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
			HostID: h.Host.ID, AdamID: app.AdamID, CommandUUID: cmds[h.Host.ID], AssociatedEventID: "1", VPPAppTeamID: app.AppTeamID,
		}))
	}

	// Android installs bypass the upcoming queue, so every unfinished row counts as pending.
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 3}, vppSplitSummary(t, ds, app))
	for _, h := range []*fleet.AndroidHost{failedHost, installedHost, pendingHost} {
		status, _ := vppSplitStatus(t, ds, h.Host, app.AdamID)
		require.Equal(t, fleet.SoftwareInstallPending, status)
	}
	require.ElementsMatch(t, []uint{failedHost.Host.ID, installedHost.Host.ID, pendingHost.Host.ID},
		vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstallPending))
	require.Empty(t, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstalled))

	require.NoError(t, ds.BulkSetVPPInstallsAsFailed(ctx, failedHost.Host.ID, []string{cmds[failedHost.Host.ID]}))
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, installedHost.Host.ID, []string{cmds[installedHost.Host.ID]}))

	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1, Failed: 1, Installed: 1}, vppSplitSummary(t, ds, app))
	for hostID, want := range map[*fleet.AndroidHost]fleet.SoftwareInstallerStatus{
		failedHost: fleet.SoftwareInstallFailed, installedHost: fleet.SoftwareInstalled, pendingHost: fleet.SoftwareInstallPending,
	} {
		status, _ := vppSplitStatus(t, ds, hostID.Host, app.AdamID)
		require.Equal(t, want, status)
	}
	require.Equal(t, []uint{pendingHost.Host.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstallPending))
	require.Equal(t, []uint{failedHost.Host.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstallFailed))
	require.Equal(t, []uint{installedHost.Host.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstalled))
}

func testVPPSplitAppleVerificationTimeout(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	vppApp := vppSplitApp(t, ds, "split_timeout_vpp", fleet.IOSPlatform, nil)
	verifiedApp := vppSplitApp(t, ds, "split_timeout_verified", fleet.IOSPlatform, nil)
	queuedApp := vppSplitApp(t, ds, "split_timeout_queued", fleet.IOSPlatform, nil)
	androidApp := vppSplitApp(t, ds, "com.split.timeout", fleet.AndroidPlatform, nil)
	inHouseID, _ := vppSplitInHouseApp(t, ds, user, "com.split.timeout.inhouse")
	inHouseTitle := func() uint {
		var id uint
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT title_id FROM in_house_apps WHERE id = ?`, inHouseID)
		})
		return id
	}()

	host := vppSplitHost(t, ds, "timeout-ios", "ios", nil)
	pendingCmd := vppSplitQueue(t, ds, host, vppApp, fleet.HostSoftwareInstallOptions{})
	_ = pendingCmd
	// Queued behind the pending install, so no install row exists for it.
	vppSplitQueue(t, ds, host, queuedApp, fleet.HostSoftwareInstallOptions{})
	inHouseHost := vppSplitHost(t, ds, "timeout-inhouse", "ios", nil)
	require.NoError(t, ds.InsertHostInHouseAppInstall(ctx, inHouseHost.ID, inHouseID, inHouseTitle, uuid.NewString(), fleet.HostSoftwareInstallOptions{}))

	verifiedHost := vppSplitHost(t, ds, "timeout-verified", "ios", nil)
	vppSplitVerify(t, ds, verifiedHost, vppSplitQueue(t, ds, verifiedHost, verifiedApp, fleet.HostSoftwareInstallOptions{}))

	androidHost, err := ds.NewAndroidHost(ctx, createAndroidHost(uuid.NewString()), false)
	require.NoError(t, err)
	require.NoError(t, ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
		HostID: androidHost.Host.ID, AdamID: androidApp.AdamID, CommandUUID: uuid.NewString(), AssociatedEventID: "1", VPPAppTeamID: androidApp.AppTeamID,
	}))

	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1}, vppSplitSummary(t, ds, vppApp))
	inHouseSummary, err := ds.GetSummaryHostInHouseAppInstalls(ctx, nil, inHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1}, *inHouseSummary)

	require.NoError(t, ds.MarkAllPendingAppleVPPAndInHouseInstallsAsFailed(ctx, "vpp_split_test_job"))

	// Pending Apple installs of both kinds fail and leave the upcoming queue.
	require.Equal(t, fleet.VPPAppStatusSummary{Failed: 1}, vppSplitSummary(t, ds, vppApp))
	inHouseSummary, err = ds.GetSummaryHostInHouseAppInstalls(ctx, nil, inHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Failed: 1}, *inHouseSummary)
	status, _ := vppSplitStatus(t, ds, host, vppApp.AdamID)
	require.Equal(t, fleet.SoftwareInstallFailed, status)
	require.Equal(t, 0, vppSplitCount(t, ds, `SELECT COUNT(*) FROM upcoming_activities ua JOIN vpp_app_upcoming_activities v ON v.upcoming_activity_id = ua.id WHERE v.adam_id = ?`, vppApp.AdamID))

	// Verified installs, Android installs and queued installs without an install row are untouched.
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, verifiedApp))
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1}, vppSplitSummary(t, ds, androidApp))
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1}, vppSplitSummary(t, ds, queuedApp))
}

func testVPPSplitAndroidVerificationTimeout(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	appA := vppSplitApp(t, ds, "com.split.timeout.a", fleet.AndroidPlatform, nil)
	appB := vppSplitApp(t, ds, "com.split.timeout.b", fleet.AndroidPlatform, nil)
	iosApp := vppSplitApp(t, ds, "split_timeout_ios", fleet.IOSPlatform, nil)

	hostA, err := ds.NewAndroidHost(ctx, createAndroidHost(uuid.NewString()), false)
	require.NoError(t, err)
	hostB, err := ds.NewAndroidHost(ctx, createAndroidHost(uuid.NewString()), false)
	require.NoError(t, err)

	insert := func(host *fleet.AndroidHost, app *fleet.VPPApp) string {
		cmd := uuid.NewString()
		require.NoError(t, ds.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
			HostID: host.Host.ID, AdamID: app.AdamID, CommandUUID: cmd, AssociatedEventID: "1", VPPAppTeamID: app.AppTeamID,
		}))
		return cmd
	}
	insert(hostA, appA)
	installedCmd := insert(hostA, appB)
	insert(hostB, appA)
	require.NoError(t, ds.BulkSetVPPInstallsAsVerified(ctx, hostA.Host.ID, []string{installedCmd}))

	iosHost := vppSplitHost(t, ds, "timeout-android-ios", "ios", nil)
	vppSplitQueue(t, ds, iosHost, iosApp, fleet.HostSoftwareInstallOptions{})

	// The per-host variant fails only that host's pending installs and returns one activity per
	// failed install.
	users, activities, err := ds.MarkAllPendingVPPInstallsAsFailedForAndroidHost(ctx, hostA.Host.ID)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Len(t, activities, 1)
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1, Failed: 1}, vppSplitSummary(t, ds, appA))
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, appB))

	// A duplicate delivery finds nothing pending.
	_, activities, err = ds.MarkAllPendingVPPInstallsAsFailedForAndroidHost(ctx, hostA.Host.ID)
	require.NoError(t, err)
	require.Empty(t, activities)

	// The enterprise-wide variant fails every pending Android install and leaves Apple ones.
	require.NoError(t, ds.MarkAllPendingAndroidVPPInstallsAsFailed(ctx))
	require.Equal(t, fleet.VPPAppStatusSummary{Failed: 2}, vppSplitSummary(t, ds, appA))
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, appB))
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1}, vppSplitSummary(t, ds, iosApp))
	require.ElementsMatch(t, []uint{hostA.Host.ID, hostB.Host.ID}, vppSplitHostsByStatus(t, ds, appA, fleet.SoftwareInstallFailed))
}

func testVPPSplitCancelActivated(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	app := vppSplitApp(t, ds, "split_cancel", fleet.MacOSPlatform, nil)
	host := vppSplitHost(t, ds, "cancel-host", "darwin", nil)
	inHouseID, inHouseTitle := vppSplitInHouseApp(t, ds, user, "com.split.cancel.inhouse")
	iosHost := vppSplitHost(t, ds, "cancel-ios", "ios", nil)

	// A prior verified install, then an activated reinstall waiting on verification.
	vppSplitVerify(t, ds, host, vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{}))
	cmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{ForSetupExperience: true})
	createVPPAppInstallResult(t, ds, host, cmd, fleet.MDMAppleStatusAcknowledged)
	require.NoError(t, ds.AssociateMDMInstallToVerificationUUID(ctx, cmd, uuid.NewString(), host.UUID))
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, name, status, nano_command_uuid) VALUES (?, ?, ?, ?)`,
			host.UUID, app.Name, fleet.SetupExperienceStatusRunning, cmd)
		return err
	})
	hostCommands := func() int {
		return vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_mdm_commands WHERE host_id = ? AND command_type = ?`, host.ID, fleet.VerifySoftwareInstallVPPPrefix)
	}
	require.Equal(t, 1, hostCommands())
	status, _ := vppSplitStatus(t, ds, host, app.AdamID)
	require.Equal(t, fleet.SoftwareInstallPending, status)

	act, err := ds.CancelHostUpcomingActivity(ctx, host.ID, cmd)
	require.NoError(t, err)
	require.IsType(t, fleet.ActivityTypeCanceledInstallAppStoreApp{}, act)

	// The list falls back to the prior install and the summary ignores the canceled one.
	status, _ = vppSplitStatus(t, ds, host, app.AdamID)
	require.Equal(t, fleet.SoftwareInstalled, status)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, app))
	require.Equal(t, []uint{host.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstalled))
	require.Empty(t, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstallPending))
	require.Equal(t, 0, hostCommands())
	var stepStatus fleet.SetupExperienceStatusResultStatus
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &stepStatus, `SELECT status FROM setup_experience_status_results WHERE host_uuid = ?`, host.UUID)
	})
	require.Equal(t, fleet.SetupExperienceStatusFailure, stepStatus)

	// A reinstall after the cancel is a new current install.
	reinstall := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{})
	status, _ = vppSplitStatus(t, ds, host, app.AdamID)
	require.Equal(t, fleet.SoftwareInstallPending, status)
	vppSplitVerify(t, ds, host, reinstall)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, app))

	// In-house installs cancel the same way.
	inHouseCmd := uuid.NewString()
	require.NoError(t, ds.InsertHostInHouseAppInstall(ctx, iosHost.ID, inHouseID, inHouseTitle, inHouseCmd, fleet.HostSoftwareInstallOptions{}))
	summary, err := ds.GetSummaryHostInHouseAppInstalls(ctx, nil, inHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Pending: 1}, *summary)
	_, err = ds.CancelHostUpcomingActivity(ctx, iosHost.ID, inHouseCmd)
	require.NoError(t, err)
	summary, err = ds.GetSummaryHostInHouseAppInstalls(ctx, nil, inHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{}, *summary)
}

func testVPPSplitHostDelete(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	app := vppSplitApp(t, ds, "split_delete", fleet.IOSPlatform, nil)
	inHouseID, inHouseTitle := vppSplitInHouseApp(t, ds, user, "com.split.delete.inhouse")

	seed := func(name string) *fleet.Host {
		host := vppSplitHost(t, ds, name, "ios", nil)
		vppSplitVerify(t, ds, host, vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{}))
		require.NoError(t, ds.InsertHostInHouseAppInstall(ctx, host.ID, inHouseID, inHouseTitle, uuid.NewString(), fleet.HostSoftwareInstallOptions{}))
		return host
	}
	single, batchA, batchB, survivor := seed("delete-single"), seed("delete-a"), seed("delete-b"), seed("delete-survivor")

	rows := func(table string, hostID uint) int {
		return vppSplitCount(t, ds, `SELECT COUNT(*) FROM `+table+` WHERE host_id = ?`, hostID)
	}
	for _, h := range []*fleet.Host{single, batchA, batchB, survivor} {
		require.Equal(t, 1, rows("host_vpp_software_installs", h.ID))
		require.Equal(t, 1, rows("host_in_house_software_installs", h.ID))
	}
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 4}, vppSplitSummary(t, ds, app))

	require.NoError(t, ds.DeleteHost(ctx, single.ID))
	require.NoError(t, ds.DeleteHosts(ctx, []uint{batchA.ID, batchB.ID}))

	for _, h := range []*fleet.Host{single, batchA, batchB} {
		require.Zero(t, rows("host_vpp_software_installs", h.ID))
		require.Zero(t, rows("host_in_house_software_installs", h.ID))
	}
	require.Equal(t, 1, rows("host_vpp_software_installs", survivor.ID))
	require.Equal(t, 1, rows("host_in_house_software_installs", survivor.ID))
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, app))
	require.Equal(t, []uint{survivor.ID}, vppSplitHostsByStatus(t, ds, app, fleet.SoftwareInstalled))

	// New tables that carry install state keyed by host must be cleaned up too. Skipped before they exist.
	if vppSplitCount(t, ds, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'host_vpp_app_installs'`) == 1 {
		for _, h := range []*fleet.Host{single, batchA, batchB} {
			require.Zero(t, rows("host_vpp_app_installs", h.ID))
		}
		require.Equal(t, 1, rows("host_vpp_app_installs", survivor.ID))
	}
}

func testVPPSplitFleetTransfer(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	fleet1, err := ds.NewTeam(ctx, &fleet.Team{Name: "transfer 1"})
	require.NoError(t, err)
	fleet2, err := ds.NewTeam(ctx, &fleet.Team{Name: "transfer 2"})
	require.NoError(t, err)

	inBoth := vppSplitApp(t, ds, "split_transfer_both", fleet.MacOSPlatform, &fleet1.ID)
	inBothOther := vppSplitApp(t, ds, "split_transfer_both", fleet.MacOSPlatform, &fleet2.ID)
	onlyFirst := vppSplitApp(t, ds, "split_transfer_first", fleet.MacOSPlatform, &fleet1.ID)
	require.NotEqual(t, inBoth.AppTeamID, inBothOther.AppTeamID)

	host := vppSplitHost(t, ds, "transfer-host", "darwin", &fleet1.ID)
	vppSplitVerify(t, ds, host, vppSplitQueue(t, ds, host, inBoth, fleet.HostSoftwareInstallOptions{}))
	vppSplitVerify(t, ds, host, vppSplitQueue(t, ds, host, onlyFirst, fleet.HostSoftwareInstallOptions{}))
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, inBoth))

	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&fleet2.ID, []uint{host.ID})))
	host.TeamID = &fleet2.ID

	// The install is shown under the other fleet's version of the same app.
	status, listed := vppSplitStatus(t, ds, host, inBoth.AdamID)
	require.True(t, listed)
	require.Equal(t, fleet.SoftwareInstalled, status)
	titles, _, err := ds.ListHostSoftware(ctx, host, fleet.HostSoftwareTitleListOptions{IncludeAvailableForInstall: true})
	require.NoError(t, err)
	var found bool
	for _, title := range titles {
		if title.AppStoreApp != nil && title.AppStoreApp.AppStoreID == inBoth.AdamID {
			found = true
			require.NotNil(t, title.AppStoreApp.LastInstall)
		}
	}
	require.True(t, found)

	// The summaries only count hosts in the version's fleet, and the old version no longer counts the host.
	require.Equal(t, fleet.VPPAppStatusSummary{}, vppSplitSummary(t, ds, inBoth))
	// Hosts by status looks at the app regardless of the fleet.
	require.Equal(t, []uint{host.ID}, vppSplitHostsByStatus(t, ds, inBoth, fleet.SoftwareInstalled, fleet2.ID))

	// An app that doesn't exist in the new fleet disappears from the list.
	_, listed = vppSplitStatus(t, ds, host, onlyFirst.AdamID)
	require.False(t, listed)

	// Back to the first fleet, both installs show again.
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&fleet1.ID, []uint{host.ID})))
	host.TeamID = &fleet1.ID
	for _, adamID := range []string{inBoth.AdamID, onlyFirst.AdamID} {
		status, listed = vppSplitStatus(t, ds, host, adamID)
		require.True(t, listed)
		require.Equal(t, fleet.SoftwareInstalled, status)
	}
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, inBoth))
}

func testVPPSplitTokenRemoval(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	tok := test.CreateInsertGlobalVPPToken(t, ds)
	require.NotNil(t, tok)
	app := vppSplitApp(t, ds, "split_token", fleet.MacOSPlatform, nil)
	host := vppSplitHost(t, ds, "token-host", "darwin", nil)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE vpp_apps_teams SET vpp_token_id = ? WHERE id = ?`, tok.ID, app.AppTeamID)
		return err
	})
	installedCmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{VPPTokenID: tok.ID})
	vppSplitVerify(t, ds, host, installedCmd)

	linked := func() int {
		return vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ? AND vpp_token_id IS NOT NULL AND vpp_app_team_id IS NOT NULL`, host.ID)
	}
	require.Equal(t, 1, linked())

	require.NoError(t, ds.DeleteVPPToken(ctx, tok.ID))

	// The install row survives unlinked from the version and the token, and still resolves.
	require.Equal(t, 1, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ?`, host.ID))
	require.Zero(t, linked())
	install, err := ds.GetHostVPPInstallByCommandUUID(ctx, installedCmd)
	require.NoError(t, err)
	require.NotNil(t, install)
	_, listed := vppSplitStatus(t, ds, host, app.AdamID)
	require.False(t, listed) // the app is no longer available in the fleet

	// Adding the app back under a new token picks the old install up through the unlinked row.
	test.CreateInsertGlobalVPPToken(t, ds)
	readded := vppSplitApp(t, ds, "split_token", fleet.MacOSPlatform, nil)
	require.NotEqual(t, app.AppTeamID, readded.AppTeamID)
	status, listed := vppSplitStatus(t, ds, host, app.AdamID)
	require.True(t, listed)
	require.Equal(t, fleet.SoftwareInstalled, status)
	// Unlinked installs only supersede older failures in the summary, they aren't counted.
	require.Equal(t, fleet.VPPAppStatusSummary{}, vppSplitSummary(t, ds, readded))
	require.Equal(t, []uint{host.ID}, vppSplitHostsByStatus(t, ds, readded, fleet.SoftwareInstalled))

	// A newer install of the re-added version supersedes the unlinked one.
	vppSplitVerify(t, ds, host, vppSplitQueue(t, ds, host, readded, fleet.HostSoftwareInstallOptions{}))
	require.Equal(t, 1, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ? AND vpp_app_team_id = ?`, host.ID, readded.AppTeamID))
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, readded))
}

func testVPPSplitAppDeletionCascade(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	app := vppSplitApp(t, ds, "split_cascade", fleet.MacOSPlatform, nil)
	host := vppSplitHost(t, ds, "cascade-host", "darwin", nil)
	cmd := vppSplitQueue(t, ds, host, app, fleet.HostSoftwareInstallOptions{})
	vppSplitVerify(t, ds, host, cmd)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1}, vppSplitSummary(t, ds, app))

	// Removing the app from the fleet keeps the install row, unlinked from the version.
	require.NoError(t, ds.DeleteVPPAppFromTeam(ctx, nil, app.VPPAppID, nil))
	require.Equal(t, 1, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE command_uuid = ? AND vpp_app_team_id IS NULL`, cmd))
	install, err := ds.GetHostVPPInstallByCommandUUID(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, install)

	// Deleting the catalog row erases its install history.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM vpp_apps WHERE adam_id = ? AND platform = ?`, app.AdamID, app.Platform)
		return err
	})
	require.Zero(t, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ?`, host.ID))
	install, err = ds.GetHostVPPInstallByCommandUUID(ctx, cmd)
	require.NoError(t, err)
	require.Nil(t, install)

	// Same for in-house apps, whose installs cascade from the app row.
	iosHost := vppSplitHost(t, ds, "cascade-ios", "ios", nil)
	inHouseID, inHouseTitle := vppSplitInHouseApp(t, ds, user, "com.split.cascade.inhouse")
	require.NoError(t, ds.InsertHostInHouseAppInstall(ctx, iosHost.ID, inHouseID, inHouseTitle, uuid.NewString(), fleet.HostSoftwareInstallOptions{}))
	require.Equal(t, 1, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_in_house_software_installs WHERE in_house_app_id = ?`, inHouseID))
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM in_house_apps WHERE id = ?`, inHouseID)
		return err
	})
	require.Zero(t, vppSplitCount(t, ds, `SELECT COUNT(*) FROM host_in_house_software_installs WHERE host_id = ?`, iosHost.ID))
}

func testVPPSplitNanoOrphanGuard(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay
	inHouseID, _ := vppSplitInHouseApp(t, ds, user, "com.split.orphan.inhouse")

	// Old commands whose only reference is an install row survive the orphan walk, and an
	// unreferenced one doesn't.
	vppRef := f.orphan("InstallApplication", 2*day)
	f.exec(`INSERT INTO vpp_apps (adam_id, platform, name, latest_version) VALUES ('split-orphan', 'ios', 'App', '1.0')`)
	f.exec(`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'split-orphan', 'ios', ?)`, f.hostID, vppRef)
	inHouseRef := f.orphan("InstallEnterpriseApplication", 2*day)
	f.exec(`INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, command_uuid, platform) VALUES (?, ?, ?, 'ios')`, f.hostID, inHouseID, inHouseRef)
	unreferenced := f.orphan("InstallApplication", 2*day)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{OrphanCommandsDeleted: 1}, stats)
	require.Zero(t, f.commandRows(unreferenced))
	require.Equal(t, 1, f.commandRows(vppRef))
	require.Equal(t, 1, f.commandRows(inHouseRef))

	// Once the install row is gone the command can go.
	f.exec(`DELETE FROM host_vpp_software_installs WHERE command_uuid = ?`, vppRef)
	f.exec(`DELETE FROM host_in_house_software_installs WHERE command_uuid = ?`, inHouseRef)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{OrphanCommandsDeleted: 2}, stats)
	require.Zero(t, f.commandRows(vppRef))
	require.Zero(t, f.commandRows(inHouseRef))
}

func testVPPSplitFailedInstallWindow(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	app := vppSplitApp(t, ds, "split_window", fleet.MacOSPlatform, nil)
	otherVersion, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             app.Name,
		BundleIdentifier: app.BundleIdentifier,
		AdamID:           app.AdamID,
		Platform:         app.Platform,
		VersionName:      "Other",
	}, nil, nil)
	require.NoError(t, err)
	require.NotEqual(t, app.AppTeamID, otherVersion.AppTeamID)

	failed := vppSplitHost(t, ds, "window-failed", "darwin", nil)
	retried := vppSplitHost(t, ds, "window-retried", "darwin", nil)
	recovered := vppSplitHost(t, ds, "window-recovered", "darwin", nil)
	otherVersionHost := vppSplitHost(t, ds, "window-other-version", "darwin", nil)
	clean := vppSplitHost(t, ds, "window-clean", "darwin", nil)
	hostIDs := []uint{failed.ID, retried.ID, recovered.ID, otherVersionHost.ID, clean.ID}

	createVPPAppInstallResult(t, ds, failed, vppSplitQueue(t, ds, failed, app, fleet.HostSoftwareInstallOptions{}), fleet.MDMAppleStatusError)

	// A retry clears the failure, so a host with a retry in progress no longer counts.
	retriedCmd := vppSplitQueue(t, ds, retried, app, fleet.HostSoftwareInstallOptions{})
	createVPPAppInstallResult(t, ds, retried, retriedCmd, fleet.MDMAppleStatusError)
	vppSplitRetry(t, ds, retriedCmd)

	// A failure followed by a verified install still counts, the read doesn't look at later installs.
	createVPPAppInstallResult(t, ds, recovered, vppSplitQueue(t, ds, recovered, app, fleet.HostSoftwareInstallOptions{}), fleet.MDMAppleStatusError)
	vppSplitVerify(t, ds, recovered, vppSplitQueue(t, ds, recovered, app, fleet.HostSoftwareInstallOptions{}))

	createVPPAppInstallResult(t, ds, otherVersionHost, vppSplitQueue(t, ds, otherVersionHost, otherVersion, fleet.HostSoftwareInstallOptions{}), fleet.MDMAppleStatusError)
	vppSplitVerify(t, ds, clean, vppSplitQueue(t, ds, clean, app, fleet.HostSoftwareInstallOptions{}))

	got, err := ds.GetHostIDsWithRecentFailedVPPAppInstall(ctx, app.AppTeamID, time.Now().Add(-time.Hour), hostIDs)
	require.NoError(t, err)
	require.Equal(t, map[uint]struct{}{failed.ID: {}, recovered.ID: {}}, got)

	got, err = ds.GetHostIDsWithRecentFailedVPPAppInstall(ctx, otherVersion.AppTeamID, time.Now().Add(-time.Hour), hostIDs)
	require.NoError(t, err)
	require.Equal(t, map[uint]struct{}{otherVersionHost.ID: {}}, got)

	// Installs created before the window don't count.
	got, err = ds.GetHostIDsWithRecentFailedVPPAppInstall(ctx, app.AppTeamID, time.Now().Add(time.Hour), hostIDs)
	require.NoError(t, err)
	require.Empty(t, got)

	// Only the given hosts are considered.
	got, err = ds.GetHostIDsWithRecentFailedVPPAppInstall(ctx, app.AppTeamID, time.Now().Add(-time.Hour), []uint{recovered.ID, clean.ID})
	require.NoError(t, err)
	require.Equal(t, map[uint]struct{}{recovered.ID: {}}, got)
}
