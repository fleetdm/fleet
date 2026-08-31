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
		{"CoveredApps", testPatchNotificationCoveredApps},
		{"UnsentForHost", testUnsentPatchNotificationForHost},
		{"AddAndListApps", testPatchNotificationAddAndListApps},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// newPatchNotification inserts a patch notification for a host in the given
// delivery status, straight into the tables. Fleet has no endpoint for creating
// one: it is always the server that decides an end user needs telling.
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

func testPatchNotificationCoveredApps(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "covered-host", "", "covered-key", "covered-uuid", time.Now())

	// Each status gets its own app so one pass can tell which ones still count.
	// A notification that failed, expired, or was acted on no longer speaks for
	// its app, so the app needs telling about again.
	stillCovers := map[string]bool{
		notifications_api.EndUserNotificationPending:    true,
		notifications_api.EndUserNotificationDispatched: true,
		notifications_api.EndUserNotificationFailed:     false,
		notifications_api.EndUserNotificationExpired:    false,
		notifications_api.EndUserNotificationActed:      false,
	}

	var titleID uint = 100
	for status, wantCovered := range stillCovers {
		titleID++
		notificationUUID := newPatchNotification(t, ds, host.ID, status, 0)
		require.NoError(t, ds.AddPatchNotificationApp(ctx, notificationUUID,
			fleet.PatchNotificationApp{SoftwareTitleID: titleID}))

		covered, err := ds.PatchNotificationCoversApp(ctx, host.ID, titleID)
		require.NoError(t, err)
		assert.Equal(t, wantCovered, covered, "status %s", status)

		// Another host's notification doesn't cover this host's apps.
		otherHost := test.NewHost(t, ds, "other-host-"+status, "", "key-"+status, "uuid-"+status, time.Now())
		covered, err = ds.PatchNotificationCoversApp(ctx, otherHost.ID, titleID)
		require.NoError(t, err)
		assert.False(t, covered)
	}

	// An app no notification lists at all.
	covered, err := ds.PatchNotificationCoversApp(ctx, host.ID, 999)
	require.NoError(t, err)
	assert.False(t, covered)
}

func testUnsentPatchNotificationForHost(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "unsent-host", "", "unsent-key", "unsent-uuid", time.Now())

	notificationUUID, err := ds.UnsentPatchNotificationForHost(ctx, host.ID)
	require.NoError(t, err)
	assert.Empty(t, notificationUUID)

	pendingUUID := newPatchNotification(t, ds, host.ID, notifications_api.EndUserNotificationPending, 0)
	notificationUUID, err = ds.UnsentPatchNotificationForHost(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, pendingUUID, notificationUUID)

	// A notification that went out and came back pending is on a schedule the end
	// user has already seen, so a new app doesn't belong on it.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE notifications_end_user SET attempt_count = 1 WHERE uuid = ?`, pendingUUID)
		return err
	})
	notificationUUID, err = ds.UnsentPatchNotificationForHost(ctx, host.ID)
	require.NoError(t, err)
	assert.Empty(t, notificationUUID)
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

	// A skip that races another for the same app adds nothing rather than failing.
	require.NoError(t, ds.AddPatchNotificationApp(ctx, notificationUUID, app))

	apps, err := ds.ListPatchNotificationApps(ctx, notificationUUID)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, titleID, apps[0].SoftwareTitleID)
	require.NotNil(t, apps[0].PolicyID)
	assert.Equal(t, policy.ID, *apps[0].PolicyID)
	require.NotNil(t, apps[0].SoftwareInstallerID)
	assert.Equal(t, installerID, *apps[0].SoftwareInstallerID)
	assert.Equal(t, "Notified App", apps[0].Name)
	// Falls back to the title name until an admin sets one for the fleet.
	assert.Equal(t, "Notified App", apps[0].DisplayName)
	assert.False(t, apps[0].HasIcon)

	// The name the end user sees is the one set for their host's fleet.
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
}
