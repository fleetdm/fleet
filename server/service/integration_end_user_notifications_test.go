package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const endUserNotificationColumns = `
	id, uuid, host_id, status, kind, payload, attempt_count, next_attempt_at,
	displayed_at, execution_id, last_exit_code, last_reason, expires_at,
	created_at, updated_at
`

// newTestNotification inserts a notification row directly, bypassing the
// notifications bounded context's HTTP API: there's no public endpoint for
// creating one, since Fleet itself is always the one queueing them.
func newTestNotification(t *testing.T, ds *mysql.Datastore, hostID uint, kind string, payload string) string {
	t.Helper()
	notificationUUID := uuid.NewString()
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(),
			`INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, expires_at)
				VALUES (?, ?, ?, ?, ?, NOW(6) + INTERVAL 1 HOUR)`,
			notificationUUID, hostID, notifications_api.EndUserNotificationPending, kind, json.RawMessage(payload))
		return err
	})
	return notificationUUID
}

func newRenderableTestNotification(t *testing.T, ds *mysql.Datastore, hostID uint, payload string) string {
	t.Helper()
	notificationUUID := newTestNotification(t, ds, hostID, fleet.PatchNotificationKind, payload)
	require.NoError(t, ds.NewPatchNotification(context.Background(), notificationUUID))

	titleName := uuid.NewString()
	var titleID uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(context.Background(),
			`INSERT INTO software_titles (name, source) VALUES (?, 'apps')`, titleName); err != nil {
			return err
		}
		return sqlx.GetContext(context.Background(), q, &titleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = 'apps'`, titleName)
	})
	require.NoError(t, ds.AddPatchNotificationApp(context.Background(), notificationUUID,
		fleet.PatchNotificationApp{SoftwareTitleID: titleID}))

	// so the view has an icon URL to assert on
	_, err := ds.CreateOrUpdateSoftwareTitleIcon(context.Background(), &fleet.UploadSoftwareTitleIconPayload{
		TitleID: titleID, TeamID: 0, StorageID: uuid.NewString(), Filename: "icon.png",
	})
	require.NoError(t, err)

	return notificationUUID
}

func getTestInstallAt(t *testing.T, ds *mysql.Datastore, notificationUUID string) *time.Time {
	t.Helper()
	var installAt *time.Time
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &installAt,
			`SELECT install_at FROM patch_notifications WHERE notification_uuid = ?`, notificationUUID)
	})
	return installAt
}

// setTestInstallAt moves a deadline, so a test can stand in for the hour passing.
// installAt is SQL rather than a value, since the rest of the deadline handling reads
// the database clock.
func setTestInstallAt(t *testing.T, ds *mysql.Datastore, notificationUUID string, installAt string) {
	t.Helper()
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(),
			`UPDATE patch_notifications SET install_at = `+installAt+` WHERE notification_uuid = ?`, notificationUUID)
		return err
	})
}

// markTestHostSeen puts the host inside its online window, which the deadline pass requires before
// it closes the end user's apps. The fixtures leave a host a minute behind, outside that window.
func markTestHostSeen(t *testing.T, ds *mysql.Datastore, hostID uint) {
	t.Helper()
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(),
			`UPDATE host_seen_times SET seen_time = NOW(6) WHERE host_id = ?`, hostID)
		return err
	})
}

// getTestNotification reads a notification row directly, for asserting on
// state the bounded context's own HTTP API doesn't expose (e.g. status,
// execution_id).
func getTestNotification(t *testing.T, ds *mysql.Datastore, notificationUUID string) *notifications_api.EndUserNotification {
	t.Helper()
	var got notifications_api.EndUserNotification
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &got,
			`SELECT `+endUserNotificationColumns+` FROM notifications_end_user WHERE uuid = ?`, notificationUUID)
	})
	return &got
}

func (s *integrationTestSuite) TestEndUserNotifications() {
	t := s.T()
	ctx := context.Background()

	dispatch := func(t *testing.T) {
		require.NoError(t, s.notificationsSvc.ExpireAndQueueNotifications(ctx))
	}

	// createOrbitEnrolledHost leaves the host without a device auth token.
	// Orbit sends one on check-in, and the notification URL can't be built
	// without it, so stand in for orbit here.
	newNotifiableHost := func(t *testing.T, suffix string) *fleet.Host {
		host := createOrbitEnrolledHost(t, "darwin", suffix, s.ds)
		require.NoError(t, s.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, "token-"+suffix))
		return host
	}

	// fetches the notification's script as orbit would, returning the substituted
	// script contents and the host's device token
	fetchScript := func(t *testing.T, host *fleet.Host, executionID string) (string, string) {
		var resp fleet.OrbitGetScriptResponse
		s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, executionID)),
			http.StatusOK, &resp)

		token, err := s.ds.GetDeviceAuthTokenIfFresh(ctx, host.ID, time.Hour)
		require.NoError(t, err)
		return resp.ScriptContents, token
	}

	postScriptResult := func(host *fleet.Host, executionID string, exitCode int) {
		s.Do("POST", "/api/fleet/orbit/scripts/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": %d, "output": "test", "runtime": 1}`,
				*host.OrbitNodeKey, executionID, exitCode)),
			http.StatusOK)
	}

	// stands in for Fleet Desktop closing the toast, which it does ten minutes after displaying it
	expireTestToast := func(t *testing.T, notificationUUID string) {
		t.Helper()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE notifications_end_user SET displayed_at = NOW(6) - INTERVAL 11 MINUTE WHERE uuid = ?`, notificationUUID)
			return err
		})
	}

	t.Run("dispatch queues a script and substitutes the notification URL", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-dispatch")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)

		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, dispatched.Status)
		require.NotNil(t, dispatched.ExecutionID)

		scriptContents, token := fetchScript(t, host, *dispatched.ExecutionID)
		require.NotEmpty(t, token)
		require.Contains(t, scriptContents, fmt.Sprintf("/device/%s/notifications/%s", token, notificationUUID))
		require.NotContains(t, scriptContents, "FLEET_VAR")
	})

	t.Run("orbit is told to run the notification even with scripts disabled", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-scripts-disabled")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		acr := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"server_settings": {"scripts_disabled": true}
		}`), http.StatusOK, &acr)
		require.True(t, acr.AppConfig.ServerSettings.ScriptsDisabled)
		defer func() {
			s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"server_settings": {"scripts_disabled": false}
			}`), http.StatusOK, &appConfigResponse{})
		}()

		var orbitResp fleet.OrbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
			http.StatusOK, &orbitResp)
		require.Equal(t, []string{*dispatched.ExecutionID}, orbitResp.Notifications.PendingScriptExecutionIDs)
	})

	t.Run("a notify script that exits 0 marks the notification displayed", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-exit0")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		postScriptResult(host, *dispatched.ExecutionID, 0)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, got.Status)
		require.NotNil(t, got.DisplayedAt)
		require.NotNil(t, got.LastExitCode)
		require.EqualValues(t, 0, *got.LastExitCode)
	})

	t.Run("a notify script that exits 41 on a locked screen schedules a retry", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-exit41")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		postScriptResult(host, *dispatched.ExecutionID, 41)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, got.Status)
		require.Nil(t, got.DisplayedAt)
		require.NotNil(t, got.LastReason)
		require.Equal(t, notifications_api.EndUserNotificationReasonScreenLocked, *got.LastReason)
		require.NotNil(t, got.NextAttemptAt)
	})

	t.Run("a notify script that exits 2 fails the notification with no retry", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-exit2")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		postScriptResult(host, *dispatched.ExecutionID, 2)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationFailed, got.Status)
		require.Nil(t, got.NextAttemptAt)
	})

	t.Run("a real outcome's activity appears in both the host and global feeds", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-both-feeds")
		notificationUUID := newRenderableTestNotification(t, s.ds, host.ID, `{"reminder": false}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		postScriptResult(host, *dispatched.ExecutionID, 0)

		// the notification's own script is held back from the feed, so this host's
		// only past activity is the one the outcome emitted
		var hostFeed listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &hostFeed)
		require.Len(t, hostFeed.Activities, 1)
		require.Equal(t, "notified_end_user_before_patching", hostFeed.Activities[0].Type)
		require.NotNil(t, hostFeed.Activities[0].Details)
		require.Contains(t, string(*hostFeed.Activities[0].Details), notificationUUID)

		// and nothing else has happened since, so it is the newest one globally
		var globalFeed listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &globalFeed,
			"order_key", "id", "order_direction", "desc", "per_page", "1")
		require.Len(t, globalFeed.Activities, 1)
		require.Equal(t, "notified_end_user_before_patching", globalFeed.Activities[0].Type)
		require.NotNil(t, globalFeed.Activities[0].Details)
		require.Contains(t, string(*globalFeed.Activities[0].Details), notificationUUID)
	})

	t.Run("GET returns the view the notification's kind builds", func(t *testing.T) {
		// each mode-aware logo field has a deprecated twin, and the config is rejected when the two
		// disagree, so both move together and are restored afterwards
		setOrgLogos := func(light, dark string) {
			s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
				"org_info": {
					"org_logo_url": %q,
					"org_logo_url_dark_mode": %q,
					"org_logo_url_light_background": %q,
					"org_logo_url_light_mode": %q
				}
			}`, dark, dark, light, light)), http.StatusOK, &appConfigResponse{})
		}

		var before appConfigResponse
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &before)
		defer setOrgLogos(before.AppConfig.OrgInfo.OrgLogoURLLightMode, before.AppConfig.OrgInfo.OrgLogoURLDarkMode)

		setOrgLogos("https://example.com/light.png", "https://example.com/dark.png")

		host := newNotifiableHost(t, "notif-get")
		notificationUUID := newRenderableTestNotification(t, s.ds, host.ID, `{"reminder": false}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		var resp notifications_api.NotificationView
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", token, notificationUUID),
			nil, http.StatusOK, &resp)
		require.Equal(t, notificationUUID, resp.UUID)
		require.Equal(t, "Save your work", resp.Title)
		require.Equal(t, []notifications_api.NotificationAction{
			{ID: "remind", Label: "Remind me 5 minutes before"},
			{ID: "update_now", Label: "Update now"},
		}, resp.Actions)

		// one app renders the singular form
		require.Equal(t, "This app will close and update in **1 hour**.", resp.Description)
		require.Equal(t, "https://example.com/light.png", resp.OrgLogoURLLightMode)
		require.Equal(t, "https://example.com/dark.png", resp.OrgLogoURLDarkMode)
		require.Len(t, resp.Items, 1)
		require.NotEmpty(t, resp.Items[0].Name)
		require.NotNil(t, resp.Items[0].IconURL, "the fixture gave this title an icon")
		require.Contains(t, *resp.Items[0].IconURL, token, "the icon URL carries the device token")
	})

	t.Run("GET renders the plural form for more than one app", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-get-plural")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"reminder": false}`)
		require.NoError(t, s.ds.NewPatchNotification(ctx, notificationUUID))

		for _, name := range []string{"notif-get-plural-1", "notif-get-plural-2"} {
			var titleID uint
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				if _, err := q.ExecContext(ctx,
					`INSERT INTO software_titles (name, source) VALUES (?, 'apps')`, name); err != nil {
					return err
				}
				return sqlx.GetContext(ctx, q, &titleID,
					`SELECT id FROM software_titles WHERE name = ? AND source = 'apps'`, name)
			})
			require.NoError(t, s.ds.AddPatchNotificationApp(ctx, notificationUUID,
				fleet.PatchNotificationApp{SoftwareTitleID: titleID}))
		}

		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		var resp notifications_api.NotificationView
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", token, notificationUUID),
			nil, http.StatusOK, &resp)
		require.Equal(t, "These apps will close and update in **1 hour**.", resp.Description)
		require.Len(t, resp.Items, 2)
	})

	t.Run("another host's token 404s on both endpoints", func(t *testing.T) {
		hostA := newNotifiableHost(t, "notif-cross-a")
		hostB := newNotifiableHost(t, "notif-cross-b")
		notificationUUID := newTestNotification(t, s.ds, hostA.ID, "patch", `{"title": "hello"}`)

		// dispatch to hostB too, so it has a script of its own to fetch
		otherUUID := newTestNotification(t, s.ds, hostB.ID, "patch", `{"title": "other"}`)
		dispatch(t)
		dispatchedB := getTestNotification(t, s.ds, otherUUID)
		require.NotNil(t, dispatchedB.ExecutionID)
		_, tokenB := fetchScript(t, hostB, *dispatchedB.ExecutionID)

		s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", tokenB, notificationUUID),
			nil, http.StatusNotFound)

		// acting on it is scoped the same way, and leaves hostA's notification alone
		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", tokenB, notificationUUID),
			[]byte(`{"action": "delay"}`), http.StatusNotFound)

		untouched := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, untouched.Status)
		require.Nil(t, untouched.NextAttemptAt)
	})

	t.Run("GET with an unknown uuid 404s", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-unknown-uuid")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/no-such-uuid", token),
			nil, http.StatusNotFound)
	})

	// TODO: verify is currently a stub (see apply_action.go), so this only
	// confirms the action is accepted, not that it records anything.
	t.Run("a device posting the verify action gets a 200", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-verify")
		notificationUUID := newRenderableTestNotification(t, s.ds, host.ID, `{"reminder": false}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "verify"}`), http.StatusOK)
	})

	t.Run("a delay action on a notification with no registered kind changes nothing", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-delay-unregistered")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "some_unregistered_kind", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "delay"}`), http.StatusNotFound)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, got.Status, "nothing should have applied the delay")
		require.NotNil(t, got.ExecutionID)
	})

	t.Run("GET with no kind registered 404s", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-get-unregistered")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "some_unregistered_kind", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", token, notificationUUID),
			nil, http.StatusNotFound)
	})

	t.Run("POST with an unknown action is rejected", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-bad-action")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "dance"}`), http.StatusUnprocessableEntity)
	})

	t.Run("POST with a missing action is rejected", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-no-action")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{}`), http.StatusUnprocessableEntity)
	})

	t.Run("a second due notification on the same host waits its turn", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-in-flight")
		firstUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "first"}`)
		secondUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "second"}`)

		dispatch(t)
		first := getTestNotification(t, s.ds, firstUUID)
		require.NotNil(t, first.ExecutionID, "the first notification queued for the host should dispatch")
		second := getTestNotification(t, s.ds, secondUUID)
		require.Nil(t, second.ExecutionID, "the host already has an undelivered dispatch")

		postScriptResult(host, *first.ExecutionID, 0)

		dispatch(t)
		second = getTestNotification(t, s.ds, secondUUID)
		require.Nil(t, second.ExecutionID, "the first notification is now on the host's screen")

		// stand in for the toast's ten minutes elapsing
		expireTestToast(t, firstUUID)

		dispatch(t)
		second = getTestNotification(t, s.ds, secondUUID)
		require.NotNil(t, second.ExecutionID, "the second notification should dispatch once the first is off the screen")
	})

	t.Run("canceling the notify script frees both the host and the software title for a new notification", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-cancel")
		notificationUUID := newTestNotification(t, s.ds, host.ID, fleet.PatchNotificationKind, `{"reminder": false}`)
		require.NoError(t, s.ds.NewPatchNotification(ctx, notificationUUID))

		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx,
				`INSERT INTO software_titles (name, source) VALUES ('Cancel App', 'apps')`); err != nil {
				return err
			}
			return sqlx.GetContext(ctx, q, &titleID,
				`SELECT id FROM software_titles WHERE name = 'Cancel App' AND source = 'apps'`)
		})
		require.NoError(t, s.ds.AddPatchNotificationApp(ctx, notificationUUID,
			fleet.PatchNotificationApp{SoftwareTitleID: titleID}))

		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		// an admin cancels the notify script from the host's upcoming queue
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, *dispatched.ExecutionID),
			nil, http.StatusNoContent)

		canceled := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationFailed, canceled.Status)
		require.NotNil(t, canceled.LastReason)
		require.Equal(t, notifications_api.EndUserNotificationReasonCanceled, *canceled.LastReason)

		// the next app open skip for the same software title is not dropped as already covered
		exists, err := s.ds.PatchNotificationExistsForApp(ctx, host.ID, titleID)
		require.NoError(t, err)
		require.False(t, exists)

		// and the host is no longer held behind a dispatch that never arrives
		nextUUID := newTestNotification(t, s.ds, host.ID, fleet.PatchNotificationKind, `{"reminder": false}`)
		dispatch(t)
		next := getTestNotification(t, s.ds, nextUUID)
		require.NotNil(t, next.ExecutionID, "a canceled notification should not hold up the host's queue")

		// the canceled notify script stays out of the activity feed, the same as a reported one.
		// The host has no other script in its queue, so any canceled_run_script here is this one.
		var past listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &past)
		canceledScriptActivity := (fleet.ActivityTypeCanceledRunScript{}).ActivityName()
		for _, act := range past.Activities {
			require.NotEqual(t, canceledScriptActivity, act.Type,
				"canceling a notification's script is not a canceled script run")
		}
	})

	t.Run("the queued script is in the host's upcoming queue but never its past activities", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-activities")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		// while it's queued, an admin can see it coming, attributed to Fleet
		// rather than to a person
		var upcoming listHostUpcomingActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &upcoming)
		var found *fleet.UpcomingActivity
		for _, act := range upcoming.Activities {
			if act.Details != nil && strings.Contains(string(*act.Details), *dispatched.ExecutionID) {
				found = act
			}
		}
		require.NotNil(t, found, "the notification's script should be in the host's upcoming queue")
		require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), found.Type)
		require.True(t, found.FleetInitiated)
		require.NotNil(t, found.ActorFullName)
		require.Equal(t, "Fleet", *found.ActorFullName)

		postScriptResult(host, *dispatched.ExecutionID, 0)

		var past listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &past)
		for _, act := range past.Activities {
			if act.Type == (fleet.ActivityTypeRanScript{}).ActivityName() && act.Details != nil {
				require.NotContains(t, string(*act.Details), *dispatched.ExecutionID,
					"the notification's script should not appear as a script run")
			}
		}

		// and it's out of the upcoming queue now that it's done
		upcoming = listHostUpcomingActivitiesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &upcoming)
		for _, act := range upcoming.Activities {
			if act.Details != nil {
				require.NotContains(t, string(*act.Details), *dispatched.ExecutionID)
			}
		}
	})

	t.Run("a result the notification no longer points at still leaves no past activity", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-stale-exec")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		// move execution_id on as a delay and the next dispatch would, so nothing points at the script already on the host
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE notifications_end_user SET execution_id = ? WHERE uuid = ?`,
				uuid.NewString(), notificationUUID)
			return err
		})

		postScriptResult(host, *dispatched.ExecutionID, 0)

		var past listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &past)
		for _, act := range past.Activities {
			if act.Details != nil {
				require.NotContains(t, string(*act.Details), *dispatched.ExecutionID,
					"a notification's script is still a notification's script")
			}
		}
	})

	t.Run("an internal script that isn't a notification reports normally", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-other-internal")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		// Fleet queues internal scripts for other reasons too, and their results
		// run through the same outcome recording. This one is on its own host so
		// nothing is ahead of it in the queue.
		otherHost := newNotifiableHost(t, "notif-other-internal-host")
		other, err := s.ds.NewInternalHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
			HostID:         otherHost.ID,
			ScriptContents: "echo not a notification",
		})
		require.NoError(t, err)
		postScriptResult(otherHost, other.ExecutionID, 0)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, got.Status, "the notification is untouched")
		require.Nil(t, got.DisplayedAt)
		require.Nil(t, got.LastExitCode)

		// and it keeps the past activity a notification doesn't get, so only
		// notifications are held back from the feed
		var pastResp listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", otherHost.ID), nil, http.StatusOK, &pastResp)
		require.Len(t, pastResp.Activities, 1)
		require.Equal(t, (fleet.ActivityTypeRanScript{}).ActivityName(), pastResp.Activities[0].Type)
		require.Contains(t, string(*pastResp.Activities[0].Details), other.ExecutionID)
	})

	t.Run("a host with no fresh device auth token gets no notification URL", func(t *testing.T) {
		// no SetOrUpdateDeviceAuthToken, so this host never sent one
		host := createOrbitEnrolledHost(t, "darwin", "notif-no-token", s.ds)
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		// Fleet won't mint a token on the fetch path, so the script fails to
		// resolve rather than going out with an unusable URL.
		var resp fleet.OrbitGetScriptResponse
		s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, *dispatched.ExecutionID)),
			http.StatusOK, &resp)
		require.NotNil(t, resp.ExitCode)
		require.EqualValues(t, fleet.ExitCodeFleetVarResolutionFailed, *resp.ExitCode)
		require.NotContains(t, resp.ScriptContents, "/notifications/")

		// the notification is queued up to try again rather than stuck
		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, got.Status)
		require.NotNil(t, got.NextAttemptAt)
		require.Nil(t, got.DisplayedAt)
	})

	// Fleet substitutes the host's device token into the script's URL when orbit
	// fetches the script. The script Fleet stores keeps the fleet variable, so
	// reading the script result never hands out the host's device token.
	t.Run("a script result shows the fleet variable, not the host's device token", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-script-result")
		notificationUUID := newRenderableTestNotification(t, s.ds, host.ID, `{"reminder": false}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)

		// orbit fetches the script, which is where the token is substituted
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		// exit code 50 is Fleet Desktop reporting that another notification is already displayed
		s.Do("POST", "/api/fleet/orbit/scripts/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 50, "output": "Another notification was displayed.", "runtime": 1}`,
				*host.OrbitNodeKey, *dispatched.ExecutionID)),
			http.StatusOK)

		var resp fleet.GetScriptResultResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/results/%s", *dispatched.ExecutionID),
			nil, http.StatusOK, &resp)
		require.NotNil(t, resp.ExitCode)
		require.EqualValues(t, 50, *resp.ExitCode)
		require.Equal(t, "Another notification was displayed.", resp.Output)

		require.Contains(t, resp.ScriptContents, string(fleet.FleetVarPatchNotificationURL))
		require.NotContains(t, resp.ScriptContents, token)
	})

	// the install keeps its policy but not the app open query, both while upcoming and once activated
	t.Run("update now queues an install that runs with the app open", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-update-now")
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "update-now-team"})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		installerID, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "echo", Filename: "update-now.pkg", StorageID: uuid.NewString(),
			Title: "Update Now App", Version: "1.0.0", Source: "apps", Platform: "darwin",
			UserID: s.users["admin1@example.com"].ID,
			TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
			AppOpenQuery: "SELECT 1 FROM processes WHERE name = 'app'",
		})
		require.NoError(t, err)

		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &titleID,
				`SELECT title_id FROM software_installers WHERE id = ?`, installerID)
		})

		policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
			Name: "Update Now App up to date", Query: "SELECT 1;",
		})
		require.NoError(t, err)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE policies SET notify_before_patching = 1 WHERE id = ?`, policy.ID)
			return err
		})

		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"reminder": false}`)
		require.NoError(t, s.ds.NewPatchNotification(ctx, notificationUUID))
		require.NoError(t, s.ds.AddPatchNotificationApp(ctx, notificationUUID, fleet.PatchNotificationApp{
			PolicyID: &policy.ID, SoftwareTitleID: titleID, SoftwareInstallerID: &installerID,
		}))

		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		var view notifications_api.NotificationView
		s.DoJSONWithoutAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			json.RawMessage(`{"action": "update_now"}`), http.StatusOK, &view)
		require.Len(t, view.Items, 1)
		require.Equal(t, "Updating...", view.Items[0].Status)
		require.Equal(t, []notifications_api.NotificationAction{{ID: "dismiss", Label: "Hide"}}, view.Actions)

		var queuedInstalls []struct {
			ExecutionID string `db:"execution_id"`
			PolicyID    *uint  `db:"policy_id"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &queuedInstalls, `
				SELECT ua.execution_id, siua.policy_id
				FROM upcoming_activities ua
					JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				WHERE ua.host_id = ?`, host.ID)
		})
		require.Len(t, queuedInstalls, 1, "the notification's one app gets one install")
		require.NotNil(t, queuedInstalls[0].PolicyID, "the install keeps its policy so it shows in Automation runs")
		require.Equal(t, policy.ID, *queuedInstalls[0].PolicyID)

		// the end user pressed "Update now", so this install runs with the app open
		queued, err := s.ds.GetSoftwareInstallDetails(ctx, queuedInstalls[0].ExecutionID)
		require.NoError(t, err)
		require.False(t, queued.OverridePreInstallQuery)
		require.Empty(t, queued.PreInstallCondition)

		// the install waits behind the notification's own script, so close the
		// window the way the end user would and let it activate
		postScriptResult(host, *dispatched.ExecutionID, 0)
		var activatedCount int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &activatedCount,
				`SELECT COUNT(*) FROM host_software_installs WHERE execution_id = ?`, queuedInstalls[0].ExecutionID)
		})
		require.Equal(t, 1, activatedCount, "otherwise the read below is still the queued half")

		// activating copies the decision onto the install row
		activated, err := s.ds.GetSoftwareInstallDetails(ctx, queuedInstalls[0].ExecutionID)
		require.NoError(t, err)
		require.False(t, activated.OverridePreInstallQuery)
		require.Empty(t, activated.PreInstallCondition)

		// the flags describe this install rather than the policy, and this install ran without the
		// app open gate, so neither is set even though the policy still has notify_before_patching on
		result, err := s.ds.GetSoftwareInstallResults(ctx, queuedInstalls[0].ExecutionID)
		require.NoError(t, err)
		require.False(t, result.NotifyBeforePatching)
		require.False(t, result.OverridePreInstallQuery)
	})

	// First notification, delay, reminder, then the install at install_at. A second host is included
	// so each pass handles a batch of two with separate install_at values.
	t.Run("a displayed notification is reminded and then installed at install_at", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-deadline")
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "deadline-team"})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		// a real installer and policy, so the install queued at install_at is a real one with an app
		// open query it has to ignore
		installerID, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "echo", Filename: "deadline.pkg", StorageID: uuid.NewString(),
			Title: "Deadline App", Version: "2.0.0", Source: "apps", Platform: "darwin",
			UserID: s.users["admin1@example.com"].ID,
			TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
			BundleIdentifier: "com.example.deadline",
			AppOpenQuery:     "SELECT 1 FROM processes WHERE name = 'app'",
		})
		require.NoError(t, err)

		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &titleID,
				`SELECT title_id FROM software_installers WHERE id = ?`, installerID)
		})

		policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
			Name: "Deadline App up to date", Query: "SELECT 1;",
		})
		require.NoError(t, err)

		notificationUUID := newTestNotification(t, s.ds, host.ID, fleet.PatchNotificationKind, `{"reminder": false}`)
		require.NoError(t, s.ds.NewPatchNotification(ctx, notificationUUID))
		require.NoError(t, s.ds.AddPatchNotificationApp(ctx, notificationUUID, fleet.PatchNotificationApp{
			PolicyID: &policy.ID, SoftwareTitleID: titleID, SoftwareInstallerID: &installerID,
		}))

		otherHost := newNotifiableHost(t, "notif-deadline-other")
		otherUUID := newRenderableTestNotification(t, s.ds, otherHost.ID, `{"reminder": false}`)

		queuedInstalls := func(hostID uint) []struct {
			ExecutionID string `db:"execution_id"`
			PolicyID    *uint  `db:"policy_id"`
		} {
			t.Helper()
			var queued []struct {
				ExecutionID string `db:"execution_id"`
				PolicyID    *uint  `db:"policy_id"`
			}
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.SelectContext(ctx, q, &queued, `
					SELECT ua.execution_id, siua.policy_id
					FROM upcoming_activities ua
						JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
					WHERE ua.host_id = ?`, hostID)
			})
			return queued
		}

		// install_at comes from displayed_at, so it is null until the script result arrives
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		require.Nil(t, getTestInstallAt(t, s.ds, notificationUUID), "install_at is null until the notification is displayed")
		_, token := fetchScript(t, host, *dispatched.ExecutionID)
		postScriptResult(host, *dispatched.ExecutionID, 0)

		otherDispatched := getTestNotification(t, s.ds, otherUUID)
		require.NotNil(t, otherDispatched.ExecutionID)
		postScriptResult(otherHost, *otherDispatched.ExecutionID, 0)

		// exit 0 records displayed_at, and install_at follows it an hour out
		displayed := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, displayed.DisplayedAt)
		deadline := getTestInstallAt(t, s.ds, notificationUUID)
		require.NotNil(t, deadline)
		require.WithinDuration(t, displayed.DisplayedAt.Add(time.Hour), *deadline, time.Minute)

		// the second host gets its own install_at from its own displayed_at
		otherDisplayed := getTestNotification(t, s.ds, otherUUID)
		require.NotNil(t, otherDisplayed.DisplayedAt)
		otherDeadline := getTestInstallAt(t, s.ds, otherUUID)
		require.NotNil(t, otherDeadline)
		require.WithinDuration(t, otherDisplayed.DisplayedAt.Add(time.Hour), *otherDeadline, time.Minute)

		// the end user closes the toast, which leaves the install_at just set above alone
		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "delay"}`), http.StatusOK)

		delayed := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, delayed.Status)
		require.NotNil(t, delayed.DisplayedAt, "the delay action leaves displayed_at set")
		require.JSONEq(t, `{"reminder": false}`, string(delayed.Payload))
		stillDue := getTestInstallAt(t, s.ds, notificationUUID)
		require.NotNil(t, stillDue)
		require.WithinDuration(t, *deadline, *stillDue, time.Second, "the delay action does not move install_at")

		// stand in for the hour running down. The other host keeps its hour and must be left alone.
		setTestInstallAt(t, s.ds, notificationUUID, "NOW(6) + INTERVAL 4 MINUTE")
		shortened := getTestInstallAt(t, s.ds, notificationUUID)
		require.NotNil(t, shortened)
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))

		reminded := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, reminded.Status)
		require.Nil(t, reminded.DisplayedAt, "the delay clears displayed_at so the reminder records its own")
		require.NotNil(t, reminded.LastReason)
		require.Equal(t, notifications_api.EndUserNotificationReasonDelayed, *reminded.LastReason)
		require.NotNil(t, reminded.ExecutionID, "execution_id is kept so a late script result still resolves the notification")
		require.Equal(t, *dispatched.ExecutionID, *reminded.ExecutionID)
		require.JSONEq(t, `{"reminder": false}`, string(reminded.Payload), "the delay records the 1 hour notice, and the toast picks its notice from install_at when it renders")

		untouched := getTestNotification(t, s.ds, otherUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, untouched.Status,
			"the other host's install_at is outside the reminder window")
		require.JSONEq(t, `{"reminder": false}`, string(untouched.Payload))

		// pending is the guard against a second reminder
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		require.Equal(t, notifications_api.EndUserNotificationPending, getTestNotification(t, s.ds, notificationUUID).Status)

		// the reminder goes out on the next dispatch, rendering the 5 minute copy
		dispatch(t)
		delayedAndDispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, delayedAndDispatched.ExecutionID)
		_, reminderToken := fetchScript(t, host, *delayedAndDispatched.ExecutionID)

		var view notifications_api.NotificationView
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", reminderToken, notificationUUID),
			nil, http.StatusOK, &view)
		require.Contains(t, view.Description, "**5 minutes**")
		require.Equal(t, []notifications_api.NotificationAction{
			{ID: "dismiss", Label: "Hide"},
			{ID: "update_now", Label: "Update now"},
		}, view.Actions, "the reminder swaps Remind for Hide")

		// a late reminder moves install_at out rather than losing part of its 5 minutes
		postScriptResult(host, *delayedAndDispatched.ExecutionID, 0)
		reminderDisplayed := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, reminderDisplayed.DisplayedAt)
		require.JSONEq(t, `{"reminder": true}`, string(reminderDisplayed.Payload),
			"the 5 minute toast reaching the screen is what records the reminder")
		afterReminder := getTestInstallAt(t, s.ds, notificationUUID)
		require.NotNil(t, afterReminder)
		require.False(t, afterReminder.Before(reminderDisplayed.DisplayedAt.Add(5*time.Minute)),
			"install_at is never less than 5 minutes after the reminder's displayed_at")
		require.True(t, afterReminder.After(*shortened), "install_at moved out to follow the reminder's displayed_at")
		require.WithinDuration(t, reminderDisplayed.DisplayedAt.Add(5*time.Minute), *afterReminder, time.Second)

		// one pass over both: the first is past install_at and installs, the second has just reached
		// its reminder window
		markTestHostSeen(t, s.ds, host.ID)
		setTestInstallAt(t, s.ds, notificationUUID, "NOW(6) - INTERVAL 1 MINUTE")
		setTestInstallAt(t, s.ds, otherUUID, "NOW(6) + INTERVAL 4 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))

		acted := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationActed, acted.Status,
			"the notification is acted before the install requests are queued, so update_now cannot queue them again")

		otherReminded := getTestNotification(t, s.ds, otherUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, otherReminded.Status)
		require.JSONEq(t, `{"reminder": false}`, string(otherReminded.Payload), "the delay records the 1 hour notice, and the toast picks its notice from install_at when it renders")
		require.Empty(t, queuedInstalls(otherHost.ID), "the other host's notification is delayed, not installed")

		installs := queuedInstalls(host.ID)
		require.Len(t, installs, 1, "the notification's one software title gets one install request")
		require.NotNil(t, installs[0].PolicyID, "the install request keeps its policy id")
		require.Equal(t, policy.ID, *installs[0].PolicyID)

		// no override_pre_install_query, so the app open query never runs
		queued, err := s.ds.GetSoftwareInstallDetails(ctx, installs[0].ExecutionID)
		require.NoError(t, err)
		require.False(t, queued.OverridePreInstallQuery)
		require.Empty(t, queued.PreInstallCondition)

		// acted is the guard against a second install request
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		require.Len(t, queuedInstalls(host.ID), 1)
	})

	// One policy run queues installs for two apps: the open app skips and opens a notification, the closed
	// app installs and asks for a refetch, and that refetch's policy run used to open a second notification.
	t.Run("a refetch during a patch notification does not open a second notification", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-refetch")
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "refetch-team"})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		// two patch policies, each with its own installer, the way a fleet with two out of date apps looks
		newPatchPolicy := func(t *testing.T, name string) (uint, uint) {
			installerID, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				InstallScript: "echo", Filename: name + ".pkg", StorageID: uuid.NewString(),
				Title: name, Version: "2.0.0", Source: "apps", Platform: "darwin",
				UserID: s.users["admin1@example.com"].ID,
				TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
				AppOpenQuery: "SELECT 1 FROM processes WHERE name = 'app'",
			})
			require.NoError(t, err)
			policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
				Name: name + " up to date", Query: "SELECT 1;",
			})
			require.NoError(t, err)
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `
					UPDATE policies
					SET notify_before_patching = 1, continuous_automations_enabled = 1, software_installer_id = ?
					WHERE id = ?`, installerID, policy.ID)
				return err
			})
			return policy.ID, installerID
		}
		openAppPolicyID, openAppInstallerID := newPatchPolicy(t, "Open App")
		closedAppPolicyID, closedAppInstallerID := newPatchPolicy(t, "Closed App")

		bothFailing := map[uint]*bool{openAppPolicyID: new(false), closedAppPolicyID: new(false)}

		// the install the host is running right now, since the queue activates a single install at a time
		activatedInstall := func(t *testing.T) (string, uint) {
			var activated []struct {
				ExecutionID string `db:"execution_id"`
				InstallerID uint   `db:"software_installer_id"`
			}
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.SelectContext(ctx, q, &activated, `
					SELECT ua.execution_id, siua.software_installer_id
					FROM upcoming_activities ua
						JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
					WHERE ua.host_id = ? AND ua.activated_at IS NOT NULL`, host.ID)
			})
			require.Len(t, activated, 1)
			return activated[0].ExecutionID, activated[0].InstallerID
		}

		// an empty pre install condition output is the app open skip orbit reports
		postInstallResult := func(t *testing.T, executionID string, appWasOpen bool) {
			payload := &fleet.HostSoftwareInstallResultPayload{HostID: host.ID, InstallUUID: executionID}
			if appWasOpen {
				payload.PreInstallConditionOutput = new("")
			} else {
				payload.PreInstallConditionOutput = new("1")
				payload.InstallScriptExitCode = new(0)
				payload.InstallScriptOutput = new("ok")
			}
			s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
				OrbitNodeKey: *host.OrbitNodeKey, HostSoftwareInstallResultPayload: payload,
			}, http.StatusNoContent)
		}

		// an activated install has a row in both tables, so the union counts attempts rather than rows
		countInstalls := func(t *testing.T, installerID uint) int {
			var count int
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &count, `
					SELECT COUNT(*) FROM (
						SELECT ua.execution_id
						FROM upcoming_activities ua
							JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
						WHERE ua.host_id = ? AND siua.software_installer_id = ?
						UNION
						SELECT execution_id FROM host_software_installs
						WHERE host_id = ? AND software_installer_id = ?
					) attempts`,
					host.ID, installerID, host.ID, installerID)
			})
			return count
		}

		patchNotifications := func(t *testing.T) []string {
			var uuids []string
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.SelectContext(ctx, q, &uuids, `
					SELECT uuid FROM notifications_end_user WHERE host_id = ? AND kind = ? ORDER BY id`,
					host.ID, fleet.PatchNotificationKind)
			})
			return uuids
		}

		// run the first policy run with both policies failing, which queues an install that can skip for each app
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, bothFailing), http.StatusOK, &distributedResp)
		require.Equal(t, 1, countInstalls(t, openAppInstallerID))
		require.Equal(t, 1, countInstalls(t, closedAppInstallerID))

		// report each result as its install activates, the open app skips and the closed app installs
		for range 2 {
			executionID, installerID := activatedInstall(t)
			postInstallResult(t, executionID, installerID == openAppInstallerID)
		}

		notificationUUIDs := patchNotifications(t)
		require.Len(t, notificationUUIDs, 1, "the app that skipped opens one notification")
		apps, err := s.ds.ListPatchNotificationApps(ctx, notificationUUIDs[0])
		require.NoError(t, err)
		require.Len(t, apps, 1, "the app that installed is not notified about")
		require.NotNil(t, apps[0].SoftwareInstallerID)
		require.Equal(t, openAppInstallerID, *apps[0].SoftwareInstallerID)

		// check the successful install asked for the refetch, which is what re-runs the policies next
		refetchedHost, err := s.ds.Host(ctx, host.ID)
		require.NoError(t, err)
		require.True(t, refetchedHost.RefetchRequested)

		// queue the notification's script before the second policy run, so it activates first
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUIDs[0])
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		// run the second policy run, the one the refetch asked for, before the toast displays. The
		// notification has no deadline yet, so the policy is still free to queue another install.
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, bothFailing), http.StatusOK, &distributedResp)
		require.Equal(t, 2, countInstalls(t, openAppInstallerID), "the second policy run queues another install behind the script")
		require.Equal(t, 1, countInstalls(t, closedAppInstallerID), "the app that installed is on the continuous automation cooldown")

		postScriptResult(host, *dispatched.ExecutionID, 0)
		displayed := getTestNotification(t, s.ds, notificationUUIDs[0])
		require.NotNil(t, displayed.DisplayedAt)

		var view notifications_api.NotificationView
		s.DoJSONWithoutAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUIDs[0]),
			json.RawMessage(`{"action": "update_now"}`), http.StatusOK, &view)
		require.Equal(t, notifications_api.EndUserNotificationActed, getTestNotification(t, s.ds, notificationUUIDs[0]).Status)
		require.Equal(t, 3, countInstalls(t, openAppInstallerID), "Update now queues the install the end user asked for")

		// report the skip from the second policy run's install, which the queue reaches after Update now
		// and before the forced install
		refiredExecutionID, refiredInstallerID := activatedInstall(t)
		require.Equal(t, openAppInstallerID, refiredInstallerID)
		postInstallResult(t, refiredExecutionID, true)
		require.Equal(t, notificationUUIDs, patchNotifications(t),
			"a skip from the second policy run's install must not open a second notification")
	})

	// Fleet Desktop closes a toast ten minutes after displaying it, so a notification is on the end user's screen for the ten minutes after its displayed_at, whatever the end user pressed.

	notifyTeamAndHost := func(t *testing.T, suffix string) (*fleet.Host, uint) {
		host := newNotifiableHost(t, suffix)
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: suffix + "-team"})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
		return host, team.ID
	}

	// a patch policy with its own installer, carrying the app open query that makes the install skip
	newNotifyPatchPolicy := func(t *testing.T, teamID uint, name string) (uint, uint) {
		installerID, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "echo", Filename: name + ".pkg", StorageID: uuid.NewString(),
			Title: name, Version: "2.0.0", Source: "apps", Platform: "darwin",
			UserID: s.users["admin1@example.com"].ID,
			TeamID: &teamID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
			AppOpenQuery: "SELECT 1 FROM processes WHERE name = 'app'",
		})
		require.NoError(t, err)
		policy, err := s.ds.NewTeamPolicy(ctx, teamID, nil, fleet.PolicyPayload{
			Name: name + " up to date", Query: "SELECT 1;",
		})
		require.NoError(t, err)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				UPDATE policies
				SET notify_before_patching = 1, continuous_automations_enabled = 1, software_installer_id = ?
				WHERE id = ?`, installerID, policy.ID)
			return err
		})
		return policy.ID, installerID
	}

	// the install the host is running right now, since the queue activates a single install at a time
	activatedInstallFor := func(t *testing.T, hostID uint) (string, uint) {
		var activated []struct {
			ExecutionID string `db:"execution_id"`
			InstallerID uint   `db:"software_installer_id"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &activated, `
				SELECT ua.execution_id, siua.software_installer_id
				FROM upcoming_activities ua
					JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				WHERE ua.host_id = ? AND ua.activated_at IS NOT NULL`, hostID)
		})
		require.Len(t, activated, 1)
		return activated[0].ExecutionID, activated[0].InstallerID
	}

	// orbit reports an empty pre install condition output when it skips an install because the app is open
	postSkippedInstall := func(t *testing.T, host *fleet.Host, executionID string) {
		s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *host.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID: host.ID, InstallUUID: executionID, PreInstallConditionOutput: new(""),
			},
		}, http.StatusNoContent)
	}

	patchNotificationsFor := func(t *testing.T, hostID uint) []string {
		var uuids []string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &uuids, `
				SELECT uuid FROM notifications_end_user WHERE host_id = ? AND kind = ? ORDER BY id`,
				hostID, fleet.PatchNotificationKind)
		})
		return uuids
	}

	// runs a policy pass, lets the install skip because the app is open, and displays the notification that opens
	skipAndDisplay := func(t *testing.T, host *fleet.Host, results map[uint]*bool, wantInstallerID uint) string {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, results), http.StatusOK, &distributedResp)

		executionID, installerID := activatedInstallFor(t, host.ID)
		require.Equal(t, wantInstallerID, installerID)
		postSkippedInstall(t, host, executionID)

		uuids := patchNotificationsFor(t, host.ID)
		notificationUUID := uuids[len(uuids)-1]

		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		postScriptResult(host, *dispatched.ExecutionID, 0)
		require.NotNil(t, getTestNotification(t, s.ds, notificationUUID).DisplayedAt)

		return notificationUUID
	}

	t.Run("an app that skips while a reminder for another app is on screen gets its own notification and its own hour", func(t *testing.T) {
		host, teamID := notifyTeamAndHost(t, "notif-reminder-join")
		firstPolicyID, firstInstallerID := newNotifyPatchPolicy(t, teamID, "Reminder Hour App")
		secondPolicyID, secondInstallerID := newNotifyPatchPolicy(t, teamID, "Late Skip App")

		firstUUID := skipAndDisplay(t, host,
			map[uint]*bool{firstPolicyID: new(false), secondPolicyID: new(true)}, firstInstallerID)
		expireTestToast(t, firstUUID)

		// the first notification's hour is nearly up, so sending its reminder clears displayed_at
		setTestInstallAt(t, s.ds, firstUUID, "NOW(6) + INTERVAL 4 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		reminding := getTestNotification(t, s.ds, firstUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, reminding.Status)
		require.Nil(t, reminding.DisplayedAt)
		require.JSONEq(t, `{"reminder": false}`, string(reminding.Payload), "the delay records the 1 hour notice, and the toast picks its notice from install_at when it renders")
		firstDeadline := getTestInstallAt(t, s.ds, firstUUID)
		require.NotNil(t, firstDeadline)

		// the second app skips in that window, while the first notification is pending with no displayed_at
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, map[uint]*bool{firstPolicyID: new(true), secondPolicyID: new(false)}),
			http.StatusOK, &distributedResp)
		executionID, installerID := activatedInstallFor(t, host.ID)
		require.Equal(t, secondInstallerID, installerID)
		postSkippedInstall(t, host, executionID)

		uuids := patchNotificationsFor(t, host.ID)
		require.Len(t, uuids, 2, "the skipping app opens its own notification rather than joining the reminder")
		secondUUID := uuids[1]
		require.NotEqual(t, firstUUID, secondUUID)

		firstApps, err := s.ds.ListPatchNotificationApps(ctx, firstUUID)
		require.NoError(t, err)
		require.Len(t, firstApps, 1, "the reminder still lists only the app it was opened for")
		require.Equal(t, firstInstallerID, *firstApps[0].SoftwareInstallerID)

		secondApps, err := s.ds.ListPatchNotificationApps(ctx, secondUUID)
		require.NoError(t, err)
		require.Len(t, secondApps, 1)
		require.Equal(t, secondInstallerID, *secondApps[0].SoftwareInstallerID)

		// the new notification starts with no deadline, so it gets its own hour once displayed
		require.Nil(t, getTestInstallAt(t, s.ds, secondUUID),
			"the new notification does not inherit the reminder's deadline")
		stillDue := getTestInstallAt(t, s.ds, firstUUID)
		require.NotNil(t, stillDue)
		require.WithinDuration(t, *firstDeadline, *stillDue, time.Second)
	})

	t.Run("a notification for a second app waits while the first toast is still on screen", func(t *testing.T) {
		host, teamID := notifyTeamAndHost(t, "notif-second-app")
		firstPolicyID, firstInstallerID := newNotifyPatchPolicy(t, teamID, "First App")
		secondPolicyID, secondInstallerID := newNotifyPatchPolicy(t, teamID, "Second App")

		// the first app is out of date and open, so its install skips and the notification is displayed
		firstUUID := skipAndDisplay(t, host,
			map[uint]*bool{firstPolicyID: new(false), secondPolicyID: new(true)}, firstInstallerID)
		firstDisplayed := getTestNotification(t, s.ds, firstUUID)
		firstDeadline := getTestInstallAt(t, s.ds, firstUUID)
		require.NotNil(t, firstDeadline)

		// minutes later the second app starts failing too, and its install skips the same way
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, map[uint]*bool{firstPolicyID: new(true), secondPolicyID: new(false)}),
			http.StatusOK, &distributedResp)
		executionID, installerID := activatedInstallFor(t, host.ID)
		require.Equal(t, secondInstallerID, installerID)
		postSkippedInstall(t, host, executionID)

		// the second app gets a separate notification with its own hour, since displaying the first
		// notification closed the coalescing window
		uuids := patchNotificationsFor(t, host.ID)
		require.Len(t, uuids, 2, "an app that skips after the first toast displayed gets its own notification")
		secondUUID := uuids[1]
		require.Nil(t, getTestInstallAt(t, s.ds, secondUUID), "the second notification's hour has not started")

		dispatch(t)
		second := getTestNotification(t, s.ds, secondUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, second.Status)
		require.Nil(t, second.ExecutionID, "the second notification waits behind a toast displayed seconds ago")

		// the first toast closes itself, which is what lets the second notification display
		expireTestToast(t, firstUUID)

		dispatch(t)
		second = getTestNotification(t, s.ds, secondUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, second.Status)
		require.NotNil(t, second.ExecutionID)

		postScriptResult(host, *second.ExecutionID, 0)
		require.NotNil(t, getTestNotification(t, s.ds, secondUUID).DisplayedAt)

		// waiting does not move the first notification's hour
		first := getTestNotification(t, s.ds, firstUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, first.Status)
		require.NotNil(t, first.DisplayedAt)
		require.NotNil(t, firstDisplayed.DisplayedAt)
		stillDue := getTestInstallAt(t, s.ds, firstUUID)
		require.NotNil(t, stillDue)
		require.WithinDuration(t, *firstDeadline, *stillDue, time.Second, "the first notification keeps its original hour")
	})

	t.Run("a notification for a second app waits while Update now is showing install progress", func(t *testing.T) {
		host, teamID := notifyTeamAndHost(t, "notif-update-now-second-app")
		firstPolicyID, firstInstallerID := newNotifyPatchPolicy(t, teamID, "Update Now App")
		secondPolicyID, secondInstallerID := newNotifyPatchPolicy(t, teamID, "Later App")

		firstUUID := skipAndDisplay(t, host,
			map[uint]*bool{firstPolicyID: new(false), secondPolicyID: new(true)}, firstInstallerID)
		dispatched := getTestNotification(t, s.ds, firstUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		// the end user presses Update now. The page keeps the toast open on the Installing view, since
		// only a dismiss action posts `dismiss` over the bridge.
		var view notifications_api.NotificationView
		s.DoJSONWithoutAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, firstUUID),
			json.RawMessage(`{"action": "update_now"}`), http.StatusOK, &view)
		acted := getTestNotification(t, s.ds, firstUUID)
		require.Equal(t, notifications_api.EndUserNotificationActed, acted.Status)
		require.NotNil(t, acted.DisplayedAt, "acting on a notification leaves displayed_at set")

		// the install Update now queued runs and clears the queue, since the host activates a single install at a time
		updateNowExecutionID, updateNowInstallerID := activatedInstallFor(t, host.ID)
		require.Equal(t, firstInstallerID, updateNowInstallerID)
		s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *host.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID: host.ID, InstallUUID: updateNowExecutionID, PreInstallConditionOutput: new("1"),
				InstallScriptExitCode: new(0), InstallScriptOutput: new("ok"),
			},
		}, http.StatusNoContent)

		// the second app skips while the Installing view is still displayed
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, map[uint]*bool{firstPolicyID: new(true), secondPolicyID: new(false)}),
			http.StatusOK, &distributedResp)
		executionID, installerID := activatedInstallFor(t, host.ID)
		require.Equal(t, secondInstallerID, installerID)
		postSkippedInstall(t, host, executionID)

		uuids := patchNotificationsFor(t, host.ID)
		require.Len(t, uuids, 2)
		secondUUID := uuids[1]

		// acted ends the countdown but leaves the toast displayed polling each app's install status, so it
		// still blocks a new notification
		dispatch(t)
		second := getTestNotification(t, s.ds, secondUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, second.Status)
		require.Nil(t, second.ExecutionID, "the second notification waits behind the Installing view")
		require.Equal(t, notifications_api.EndUserNotificationActed, getTestNotification(t, s.ds, firstUUID).Status)

		expireTestToast(t, firstUUID)

		dispatch(t)
		second = getTestNotification(t, s.ds, secondUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, second.Status)
		require.NotNil(t, second.ExecutionID, "an acted notification stops blocking once its toast closes")
	})

	t.Run("two reminders on the same host are displayed in sequence rather than replacing each other", func(t *testing.T) {
		host, teamID := notifyTeamAndHost(t, "notif-two-reminders")
		firstPolicyID, firstInstallerID := newNotifyPatchPolicy(t, teamID, "Reminder One")
		secondPolicyID, secondInstallerID := newNotifyPatchPolicy(t, teamID, "Reminder Two")

		// two notifications, each with its own hour, the second displayed after the first toast closed
		firstUUID := skipAndDisplay(t, host,
			map[uint]*bool{firstPolicyID: new(false), secondPolicyID: new(true)}, firstInstallerID)
		expireTestToast(t, firstUUID)
		secondUUID := skipAndDisplay(t, host,
			map[uint]*bool{firstPolicyID: new(true), secondPolicyID: new(false)}, secondInstallerID)
		expireTestToast(t, secondUUID)
		require.NotEqual(t, firstUUID, secondUUID)

		// stand in for the first hour running down to its five minute reminder
		setTestInstallAt(t, s.ds, firstUUID, "NOW(6) + INTERVAL 4 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		firstReminder := getTestNotification(t, s.ds, firstUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, firstReminder.Status)
		require.Nil(t, firstReminder.DisplayedAt)
		require.JSONEq(t, `{"reminder": false}`, string(firstReminder.Payload), "the delay records the 1 hour notice, and the toast picks its notice from install_at when it renders")

		dispatch(t)
		require.Equal(t, notifications_api.EndUserNotificationDispatched,
			getTestNotification(t, s.ds, firstUUID).Status)

		// the second hour reaches its own reminder a couple of minutes behind the first
		setTestInstallAt(t, s.ds, secondUUID, "NOW(6) + INTERVAL 4 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		require.Equal(t, notifications_api.EndUserNotificationPending,
			getTestNotification(t, s.ds, secondUUID).Status)

		// the first reminder is dispatched and not yet displayed
		dispatch(t)
		require.Equal(t, notifications_api.EndUserNotificationPending,
			getTestNotification(t, s.ds, secondUUID).Status,
			"the second reminder waits while the first reminder has no displayed_at")

		// the first reminder is displayed, which sets its displayed_at and ends the first reason to wait
		firstDispatched := getTestNotification(t, s.ds, firstUUID)
		require.NotNil(t, firstDispatched.ExecutionID)
		postScriptResult(host, *firstDispatched.ExecutionID, 0)
		firstShown := getTestNotification(t, s.ds, firstUUID)
		require.NotNil(t, firstShown.DisplayedAt)

		// the first reminder is on the end user's screen, so the second reminder keeps waiting rather than replacing it
		dispatch(t)
		require.Equal(t, notifications_api.EndUserNotificationPending,
			getTestNotification(t, s.ds, secondUUID).Status,
			"the second reminder waits while the first reminder is on screen")

		expireTestToast(t, firstUUID)

		dispatch(t)
		secondDispatched := getTestNotification(t, s.ds, secondUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, secondDispatched.Status,
			"the second reminder is sent once the first reminder's toast has closed")
		require.NotNil(t, secondDispatched.ExecutionID)
		postScriptResult(host, *secondDispatched.ExecutionID, 0)
		secondShown := getTestNotification(t, s.ds, secondUUID)
		require.NotNil(t, secondShown.DisplayedAt, "the second reminder reaches the end user rather than being lost")
		secondDeadline := getTestInstallAt(t, s.ds, secondUUID)
		require.NotNil(t, secondDeadline)
		require.False(t, secondDeadline.Before(secondShown.DisplayedAt.Add(5*time.Minute)),
			"the second reminder gets its own five minutes, counted from when it was displayed")

		// the five minutes are up on the first notification, whose reminder had the screen to itself
		markTestHostSeen(t, s.ds, host.ID)
		setTestInstallAt(t, s.ds, firstUUID, "NOW(6) - INTERVAL 1 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		require.Equal(t, notifications_api.EndUserNotificationActed,
			getTestNotification(t, s.ds, firstUUID).Status,
			"the first app is force installed after its reminder was shown")

		var queued int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &queued, `
				SELECT COUNT(*)
				FROM upcoming_activities ua
					JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				WHERE ua.host_id = ? AND siua.software_installer_id = ?`, host.ID, firstInstallerID)
		})
		require.Equal(t, 1, queued, "the forced install is queued for the app whose reminder was shown")

		// the second notification runs down its own five minutes and installs its own app
		setTestInstallAt(t, s.ds, secondUUID, "NOW(6) - INTERVAL 1 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		require.Equal(t, notifications_api.EndUserNotificationActed,
			getTestNotification(t, s.ds, secondUUID).Status,
			"the second app is force installed after its own reminder was shown")

		var secondQueued int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &secondQueued, `
				SELECT COUNT(*)
				FROM upcoming_activities ua
					JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				WHERE ua.host_id = ? AND siua.software_installer_id = ?`, host.ID, secondInstallerID)
		})
		require.Equal(t, 1, secondQueued, "the second app gets its own forced install, not the first app's")
	})

	t.Run("a host offline through a deadline gets the 1 hour notification again each time it comes back", func(t *testing.T) {
		host, teamID := notifyTeamAndHost(t, "notif-offline-cycle")
		policyID, installerID := newNotifyPatchPolicy(t, teamID, "Offline Cycle App")

		// the 1 hour notification is displayed
		notificationUUID := skipAndDisplay(t, host, map[uint]*bool{policyID: new(false)}, installerID)
		expireTestToast(t, notificationUUID)

		// the host goes offline, so the delayed reminder waits in its queue while the deadline passes
		setTestInstallAt(t, s.ds, notificationUUID, "NOW(6) + INTERVAL 4 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		dispatch(t)
		setTestInstallAt(t, s.ds, notificationUUID, "NOW(6) - INTERVAL 1 MINUTE")

		// the host comes back and displays the 1 hour notification again
		firstReturn := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, firstReturn.ExecutionID)
		_, firstReturnToken := fetchScript(t, host, *firstReturn.ExecutionID)
		var firstReturnView notifications_api.NotificationView
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", firstReturnToken, notificationUUID),
			nil, http.StatusOK, &firstReturnView)
		require.Contains(t, firstReturnView.Description, "**1 hour**")
		postScriptResult(host, *firstReturn.ExecutionID, 0)
		firstReturnDeadline := getTestInstallAt(t, s.ds, notificationUUID)
		require.NotNil(t, firstReturnDeadline)
		require.WithinDuration(t, time.Now().UTC().Add(time.Hour), *firstReturnDeadline, time.Minute)
		expireTestToast(t, notificationUUID)

		// the new deadline comes round and the 5 minute reminder is displayed
		setTestInstallAt(t, s.ds, notificationUUID, "NOW(6) + INTERVAL 4 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		dispatch(t)
		reminder := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, reminder.ExecutionID)
		_, reminderToken := fetchScript(t, host, *reminder.ExecutionID)
		var reminderView notifications_api.NotificationView
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", reminderToken, notificationUUID),
			nil, http.StatusOK, &reminderView)
		require.Contains(t, reminderView.Description, "**5 minutes**")
		postScriptResult(host, *reminder.ExecutionID, 0)
		reminderDisplayed := getTestNotification(t, s.ds, notificationUUID)
		require.JSONEq(t, `{"reminder": true}`, string(reminderDisplayed.Payload))

		// the host goes offline before the reminder's deadline, so the deadline delays the notification instead of installing
		setTestInstallAt(t, s.ds, notificationUUID, "NOW(6) - INTERVAL 1 MINUTE")
		require.NoError(t, s.patchNotificationKind.RemindAndInstallDuePatches(ctx))
		delayedOffline := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, delayedOffline.Status)
		require.JSONEq(t, `{"reminder": false}`, string(delayedOffline.Payload))
		var queued int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &queued, `
				SELECT COUNT(*)
				FROM upcoming_activities ua
					JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				WHERE ua.host_id = ? AND siua.software_installer_id = ?`, host.ID, installerID)
		})
		require.Zero(t, queued, "nothing installs on a host that is offline at the deadline")

		// the host comes back and displays the 1 hour notification again
		dispatch(t)
		secondReturn := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, secondReturn.ExecutionID)
		_, secondReturnToken := fetchScript(t, host, *secondReturn.ExecutionID)
		var secondReturnView notifications_api.NotificationView
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", secondReturnToken, notificationUUID),
			nil, http.StatusOK, &secondReturnView)
		require.Contains(t, secondReturnView.Description, "**1 hour**")
		postScriptResult(host, *secondReturn.ExecutionID, 0)
		secondReturnDeadline := getTestInstallAt(t, s.ds, notificationUUID)
		require.NotNil(t, secondReturnDeadline)
		require.WithinDuration(t, time.Now().UTC().Add(time.Hour), *secondReturnDeadline, time.Minute)
	})
}
