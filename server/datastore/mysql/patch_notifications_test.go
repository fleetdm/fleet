package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchNotifications(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ExistsForApp", testPatchNotificationExistsForApp},
		{"AddAndListApps", testPatchNotificationAddAndListApps},
		{"ListAppsForNotifications", testPatchNotificationListAppsForNotifications},
		{"DeleteApps", testPatchNotificationDeleteApps},
		{"InstallAt", testPatchNotificationInstallAt},
		{"ListDue", testPatchNotificationListDue},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func newPatchNotification(t *testing.T, ds *Datastore, hostID uint, status string, attemptCount uint) string {
	t.Helper()
	notificationUUID := uuid.NewString()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(context.Background(), `
			INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, attempt_count, expires_at)
			VALUES (?, ?, ?, ?, '{}', ?, NOW(6) + INTERVAL 1 DAY)`,
			notificationUUID, hostID, status, fleet.PatchNotificationKind, attemptCount); err != nil {
			return err
		}
		_, err := q.ExecContext(context.Background(),
			`INSERT INTO patch_notifications (notification_uuid) VALUES (?)`, notificationUUID)
		return err
	})
	return notificationUUID
}

func newTestSoftwareTitle(t *testing.T, ds *Datastore, name string) uint {
	t.Helper()
	var titleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(context.Background(),
			`INSERT INTO software_titles (name, source) VALUES (?, 'apps')`, name); err != nil {
			return err
		}
		return sqlx.GetContext(context.Background(), q, &titleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = 'apps'`, name)
	})
	return titleID
}

func testPatchNotificationExistsForApp(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "exists-host", "", "exists-key", "exists-uuid", time.Now())

	// a pending or dispatched notification still has its apps to install, so a
	// second skip for one of those apps is dropped. A failed or expired
	// notification never installed them, and an acted notification already
	// queued the installs, so a later skip starts a new notification either way.
	stillCounts := map[string]bool{
		notifications_api.EndUserNotificationPending:    true,
		notifications_api.EndUserNotificationDispatched: true,
		notifications_api.EndUserNotificationFailed:     false,
		notifications_api.EndUserNotificationExpired:    false,
		notifications_api.EndUserNotificationActed:      false,
	}

	for status, wantExists := range stillCounts {
		titleID := newTestSoftwareTitle(t, ds, "app-"+status)
		notificationUUID := newPatchNotification(t, ds, host.ID, status, 0)
		require.NoError(t, ds.AddPatchNotificationApp(ctx, notificationUUID,
			fleet.PatchNotificationApp{SoftwareTitleID: titleID}))

		exists, err := ds.PatchNotificationExistsForApp(ctx, host.ID, titleID)
		require.NoError(t, err)
		assert.Equal(t, wantExists, exists, "status %s", status)

		// a notification is only ever for one host, so the same app on another
		// host is not listed by this notification
		otherHost := test.NewHost(t, ds, "other-host-"+status, "", "key-"+status, "uuid-"+status, time.Now())
		exists, err = ds.PatchNotificationExistsForApp(ctx, otherHost.ID, titleID)
		require.NoError(t, err)
		assert.False(t, exists)
	}

	// an app that no notification lists
	exists, err := ds.PatchNotificationExistsForApp(ctx, host.ID, newTestSoftwareTitle(t, ds, "unlisted"))
	require.NoError(t, err)
	assert.False(t, exists)
}

func testPatchNotificationAddAndListApps(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "apps-host", "", "apps-key", "apps-uuid", time.Now())
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "patch-notification-team"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	user, err := ds.NewUser(ctx, &fleet.User{
		Name: "Admin", Password: []byte("p4ssw0rd.123"), Email: "patch-notifications@example.com",
		GlobalRole: new(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "echo", Filename: "app.pkg", StorageID: uuid.NewString(),
		Title: "Notified App", Version: "1.0.0", Source: "apps", Platform: "darwin",
		UserID: user.ID, TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// MatchOrCreateSoftwareInstaller creates the software title too
	var titleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleID,
			`SELECT title_id FROM software_installers WHERE id = ?`, installerID)
	})

	policy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{Name: "notify", Query: "SELECT 1;"})
	require.NoError(t, err)

	notificationUUID := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationPending, 0)
	app := fleet.PatchNotificationApp{
		PolicyID:            &policy.ID,
		SoftwareTitleID:     titleID,
		SoftwareInstallerID: &installerID,
	}
	require.NoError(t, ds.AddPatchNotificationApp(ctx, notificationUUID, app))

	// adding the same app again does nothing rather than failing
	require.NoError(t, ds.AddPatchNotificationApp(ctx, notificationUUID, app))

	// the host's fleet has no display name or icon for this software title, so
	// the app's display name falls back to the software title's name
	apps, err := ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, titleID, apps[0].SoftwareTitleID)
	require.NotNil(t, apps[0].PolicyID)
	assert.Equal(t, policy.ID, *apps[0].PolicyID)
	require.NotNil(t, apps[0].SoftwareInstallerID)
	assert.Equal(t, installerID, *apps[0].SoftwareInstallerID)
	assert.Equal(t, "Notified App", apps[0].Name)
	assert.Equal(t, "Notified App", apps[0].DisplayName)
	assert.False(t, apps[0].HasIcon)

	// give the software title a display name and an icon in the host's fleet
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, ?)`,
			team.ID, titleID, "Notified App (renamed)")
		return err
	})
	_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
		TitleID: titleID, TeamID: team.ID, StorageID: uuid.NewString(), Filename: "icon.png",
	})
	require.NoError(t, err)

	apps, err = ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, "Notified App (renamed)", apps[0].DisplayName)
	assert.True(t, apps[0].HasIcon)

	// deleting the policy sets patch_notification_apps.policy_id to null, and the app stays listed
	_, err = ds.DeleteTeamPolicies(ctx, team.ID, []uint{policy.ID})
	require.NoError(t, err)
	apps, err = ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Nil(t, apps[0].PolicyID)

	// deleting the software title cascades and deletes the patch_notification_apps row
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM software_titles WHERE id = ?`, titleID)
		return err
	})
	apps, err = ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	assert.Empty(t, apps)

	// deleting the notification cascades and deletes the patch_notifications row
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM notifications_end_user WHERE uuid = ?`, notificationUUID)
		return err
	})
	var remaining int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &remaining,
			`SELECT COUNT(*) FROM patch_notifications WHERE notification_uuid = ?`, notificationUUID)
	})
	assert.Zero(t, remaining)
}

func testPatchNotificationListAppsForNotifications(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	installerHost := test.NewHost(t, ds, "batch-installer-host", "", "batch-installer-key", "batch-installer-uuid", time.Now())
	noInstallerHost := test.NewHost(t, ds, "batch-no-installer-host", "", "batch-no-installer-key", "batch-no-installer-uuid", time.Now())

	user, err := ds.NewUser(ctx, &fleet.User{
		Name: "Admin", Password: []byte("p4ssw0rd.123"), Email: "patch-notification-batch@example.com",
		GlobalRole: new(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	installerID, installerTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "echo", Filename: "batched.pkg", StorageID: uuid.NewString(),
		Title: "Batched App", Version: "2.0.0", Source: "apps", Platform: "darwin",
		UserID: user.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	installerNotification := newPatchNotification(t, ds, installerHost.ID, notifications_api.EndUserNotificationDispatched, 1)
	require.NoError(t, ds.AddPatchNotificationApp(ctx, installerNotification, fleet.PatchNotificationApp{
		SoftwareTitleID: installerTitleID, SoftwareInstallerID: &installerID,
	}))

	// an app whose installer was deleted still has to be listed, since Fleet has no version of it to
	// match the host against
	noInstallerTitleID := newTestSoftwareTitle(t, ds, "Uninstallable App")
	noInstallerNotification := newPatchNotification(t, ds, noInstallerHost.ID, notifications_api.EndUserNotificationDispatched, 1)
	require.NoError(t, ds.AddPatchNotificationApp(ctx, noInstallerNotification, fleet.PatchNotificationApp{
		SoftwareTitleID: noInstallerTitleID,
	}))

	// a notification listed with no apps of its own is left out of the map rather than keyed to nothing
	emptyNotification := newPatchNotification(t, ds, installerHost.ID, notifications_api.EndUserNotificationDispatched, 1)

	byNotification, err := ds.ListPatchNotificationAppsForNotifications(ctx,
		[]string{installerNotification, noInstallerNotification, emptyNotification})
	require.NoError(t, err)
	require.Len(t, byNotification, 2)
	require.NotContains(t, byNotification, emptyNotification)

	require.Len(t, byNotification[installerNotification], 1)
	installerApp := byNotification[installerNotification][0]
	assert.Equal(t, installerNotification, installerApp.NotificationUUID)
	assert.Equal(t, installerTitleID, installerApp.SoftwareTitleID)
	require.NotNil(t, installerApp.SoftwareInstallerID)
	assert.Equal(t, installerID, *installerApp.SoftwareInstallerID)
	assert.Equal(t, "2.0.0", installerApp.InstallerVersion)
	assert.False(t, installerApp.InstallQueued)

	require.Len(t, byNotification[noInstallerNotification], 1)
	noInstallerApp := byNotification[noInstallerNotification][0]
	assert.Equal(t, noInstallerNotification, noInstallerApp.NotificationUUID)
	assert.Equal(t, noInstallerTitleID, noInstallerApp.SoftwareTitleID)
	assert.Nil(t, noInstallerApp.SoftwareInstallerID)
	assert.Empty(t, noInstallerApp.InstallerVersion)

	// the flag that stops a second attempt queueing the same install twice
	require.NoError(t, ds.SetPatchNotificationAppsQueued(ctx, installerNotification, []uint{installerTitleID}))
	byNotification, err = ds.ListPatchNotificationAppsForNotifications(ctx, []string{installerNotification})
	require.NoError(t, err)
	require.Len(t, byNotification[installerNotification], 1)
	assert.True(t, byNotification[installerNotification][0].InstallQueued)

	byNotification, err = ds.ListPatchNotificationAppsForNotifications(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, byNotification)
}

func testPatchNotificationDeleteApps(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	// a host with no fleet, since ListPatchNotificationApps joins display names on COALESCE(h.team_id, 0)
	host := test.NewHost(t, ds, "delete-apps-host", "", "delete-apps-key", "delete-apps-uuid", time.Now())
	require.Nil(t, host.TeamID)

	notificationUUID := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)
	updated := newTestSoftwareTitle(t, ds, "Updated App")
	stillOpen := newTestSoftwareTitle(t, ds, "Still Open App")
	for _, titleID := range []uint{updated, stillOpen} {
		require.NoError(t, ds.AddPatchNotificationApp(ctx, notificationUUID,
			fleet.PatchNotificationApp{SoftwareTitleID: titleID}))
	}

	// deleting nothing leaves the notification's apps alone
	require.NoError(t, ds.DeletePatchNotificationApps(ctx, notificationUUID, nil))
	apps, err := ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	// the app the end user already updated stops being listed, so the reminder can't name it
	require.NoError(t, ds.DeletePatchNotificationApps(ctx, notificationUUID, []uint{updated}))
	apps, err = ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, stillOpen, apps[0].SoftwareTitleID)

	// another notification's copy of the same app is untouched
	otherUUID := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)
	require.NoError(t, ds.AddPatchNotificationApp(ctx, otherUUID,
		fleet.PatchNotificationApp{SoftwareTitleID: stillOpen}))
	require.NoError(t, ds.DeletePatchNotificationApps(ctx, notificationUUID, []uint{stillOpen}))
	apps, err = ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	assert.Empty(t, apps)
	apps, err = ds.ListPatchNotificationApps(ctx, otherUUID)
	require.NoError(t, err)
	assert.Len(t, apps, 1)
}

func testPatchNotificationInstallAt(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "install-at-host", "", "install-at-key", "install-at-uuid", time.Now())
	notificationUUID := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)

	// the first display sets the deadline
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	stored, err := ds.SetPatchNotificationInstallAt(ctx, notificationUUID, deadline)
	require.NoError(t, err)
	assert.WithinDuration(t, deadline, stored, time.Second)

	// an earlier deadline never wins, so a duplicate script result or a retry can't cut the lead
	// time the end user was promised short
	stored, err = ds.SetPatchNotificationInstallAt(ctx, notificationUUID, deadline.Add(-time.Minute))
	require.NoError(t, err)
	assert.WithinDuration(t, deadline, stored, time.Second)

	// a later one does, which is how a reminder that took a while to reach the screen still leaves
	// its full five minutes before the apps close
	pushedOut := deadline.Add(time.Minute)
	stored, err = ds.SetPatchNotificationInstallAt(ctx, notificationUUID, pushedOut)
	require.NoError(t, err)
	assert.WithinDuration(t, pushedOut, stored, time.Second)

	// an offline host's restart clears the deadline, so its next display sets a fresh one
	require.NoError(t, ds.ResetPatchNotification(ctx, notificationUUID))
	restarted := deadline.Add(-2 * time.Hour)
	stored, err = ds.SetPatchNotificationInstallAt(ctx, notificationUUID, restarted)
	require.NoError(t, err)
	assert.WithinDuration(t, restarted, stored, time.Second)

	// a notification whose patch_notifications row was never written still gets a
	// deadline, otherwise it would never be patched
	rowless := uuid.NewString()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, expires_at)
			VALUES (?, ?, ?, ?, '{}', NOW(6) + INTERVAL 1 DAY)`,
			rowless, host.ID, notifications_api.EndUserNotificationDispatched, fleet.PatchNotificationKind)
		return err
	})
	stored, err = ds.SetPatchNotificationInstallAt(ctx, rowless, deadline)
	require.NoError(t, err)
	assert.WithinDuration(t, deadline, stored, time.Second)

	// a uuid no notification has cannot hold a deadline
	_, err = ds.SetPatchNotificationInstallAt(ctx, "no-such-notification", deadline)
	require.Error(t, err)
}

func testPatchNotificationListDue(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC()

	setInstallAt := func(notificationUUID string, installAt time.Time) {
		t.Helper()
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE patch_notifications SET install_at = ? WHERE notification_uuid = ?`,
				installAt, notificationUUID)
			return err
		})
	}

	markDisplayed := func(notificationUUID string) {
		t.Helper()
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE notifications_end_user SET displayed_at = NOW(6) WHERE uuid = ?`, notificationUUID)
			return err
		})
	}

	host := test.NewHost(t, ds, "due-host", "", "due-key", "due-uuid", now)

	// 6 minutes out is outside the reminder window, 5 minutes is on its edge, and
	// the deadline itself is past due
	tooEarly := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)
	setInstallAt(tooEarly, now.Add(6*time.Minute))
	markDisplayed(tooEarly)
	inReminderWindow := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)
	setInstallAt(inReminderWindow, now.Add(4*time.Minute))
	markDisplayed(inReminderWindow)
	pastDeadline := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)
	setInstallAt(pastDeadline, now.Add(-time.Minute))
	markDisplayed(pastDeadline)

	// a notification that was never displayed has no deadline to count down
	noDeadline := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationPending, 0)

	// terminal notifications can no longer be patched, so they stay out of the batch
	terminal := make([]string, 0, 3)
	for _, status := range []string{
		notifications_api.EndUserNotificationActed,
		notifications_api.EndUserNotificationFailed,
		notifications_api.EndUserNotificationExpired,
	} {
		notificationUUID := newPatchNotification(t, ds, host.ID, status, 1)
		setInstallAt(notificationUUID, now.Add(-time.Minute))
		terminal = append(terminal, notificationUUID)
	}

	// a re-dispatch clears displayed_at, so a notice on its way to the screen still
	// has to come back: the pass sends the first notice again instead of patching
	onTheWay := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationPending, 1)
	setInstallAt(onTheWay, now.Add(-time.Minute))

	due, err := ds.ListPatchNotificationsDue(ctx, now.Add(5*time.Minute), 500)
	require.NoError(t, err)

	byUUID := make(map[string]fleet.PatchNotificationDue, len(due))
	for _, notification := range due {
		byUUID[notification.NotificationUUID] = notification
	}
	require.Len(t, byUUID, 3)
	assert.NotContains(t, byUUID, tooEarly)
	assert.NotContains(t, byUUID, noDeadline)
	for _, notificationUUID := range terminal {
		assert.NotContains(t, byUUID, notificationUUID)
	}

	require.Contains(t, byUUID, inReminderWindow)
	assert.Equal(t, host.ID, byUUID[inReminderWindow].HostID)
	assert.Equal(t, notifications_api.EndUserNotificationDispatched, byUUID[inReminderWindow].Status)
	assert.NotNil(t, byUUID[inReminderWindow].DisplayedAt)

	require.Contains(t, byUUID, pastDeadline)
	assert.NotNil(t, byUUID[pastDeadline].DisplayedAt)

	require.Contains(t, byUUID, onTheWay)
	assert.Nil(t, byUUID[onTheWay].DisplayedAt)

	// the batch is ordered by deadline, so the oldest deadline is handled first
	require.Len(t, due, 3)
	assert.True(t, due[0].InstallAt.Before(due[len(due)-1].InstallAt))

	// the limit caps the batch
	limited, err := ds.ListPatchNotificationsDue(ctx, now.Add(5*time.Minute), 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
}
