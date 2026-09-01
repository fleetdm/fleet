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
	// notification will never install its apps, so those apps are skipped into a
	// new notification instead.
	stillCounts := map[string]bool{
		notifications_api.EndUserNotificationPending:    true,
		notifications_api.EndUserNotificationDispatched: true,
		notifications_api.EndUserNotificationFailed:     false,
		notifications_api.EndUserNotificationExpired:    false,
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

	// once Update now queues the installs, the notification no longer has its  apps to install,
	// so an app whose forced install failed is skipped into a new notification
	queuedTitleID := newTestSoftwareTitle(t, ds, "app-installs-queued")
	queuedUUID := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationDispatched, 1)
	require.NoError(t, ds.AddPatchNotificationApp(ctx, queuedUUID,
		fleet.PatchNotificationApp{SoftwareTitleID: queuedTitleID}))

	exists, err = ds.PatchNotificationExistsForApp(ctx, host.ID, queuedTitleID)
	require.NoError(t, err)
	assert.True(t, exists)

	queued, err := ds.SetPatchNotificationInstallsQueued(ctx, queuedUUID)
	require.NoError(t, err)
	assert.True(t, queued)
	exists, err = ds.PatchNotificationExistsForApp(ctx, host.ID, queuedTitleID)
	require.NoError(t, err)
	assert.False(t, exists)

	// a second Update now on the same notification does not record the installs as queued a second time
	queued, err = ds.SetPatchNotificationInstallsQueued(ctx, queuedUUID)
	require.NoError(t, err)
	assert.False(t, queued)
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
	_, err = ds.GetPatchNotification(ctx, notificationUUID)
	assert.True(t, fleet.IsNotFound(err))
}
