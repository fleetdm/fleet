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
			`INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload) VALUES (?, ?, ?, ?, ?)`,
			notificationUUID, hostID, notifications_api.EndUserNotificationPending, kind, json.RawMessage(payload))
		return err
	})
	return notificationUUID
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

	t.Run("exit 0 records displayed_at", func(t *testing.T) {
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

	t.Run("exit 41 schedules a retry", func(t *testing.T) {
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

	t.Run("exit 2 is a terminal failure", func(t *testing.T) {
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

	t.Run("GET returns the notification's payload", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-get")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello world"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		var resp struct {
			Payload json.RawMessage `json:"payload"`
			Err     string          `json:"error,omitempty"`
		}
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", token, notificationUUID),
			nil, http.StatusOK, &resp)
		require.JSONEq(t, `{"title": "hello world"}`, string(resp.Payload))
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
	t.Run("POST verify is accepted", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-verify")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "verify"}`), http.StatusNoContent)
	})

	t.Run("POST delay with a registered kind delays it", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-delay-registered")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		// exit 0 is what marks it displayed, which is the point the patch kind
		// counts its next attempt from
		postScriptResult(host, *dispatched.ExecutionID, 0)
		displayed := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, displayed.DisplayedAt)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "delay"}`), http.StatusNoContent)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationPending, got.Status)
		require.NotNil(t, got.ExecutionID, "keeps its execution_id so a late result can still find it")
		require.Equal(t, *dispatched.ExecutionID, *got.ExecutionID)
		require.NotNil(t, got.LastReason)
		require.Equal(t, notifications_api.EndUserNotificationReasonDelayed, *got.LastReason)
		require.Nil(t, got.DisplayedAt, "the next send records its own display")

		// the patch kind rejoins the hour-then-five-minutes schedule rather than
		// starting a fresh wait
		require.NotNil(t, got.NextAttemptAt)
		require.WithinDuration(t, displayed.DisplayedAt.Add(55*time.Minute), *got.NextAttemptAt, time.Minute)

		// one that was delayed without ever being displayed has no mark to count
		// from, so it waits a full interval
		neverShown := newTestNotification(t, s.ds, host.ID, "patch", `{"title": "hello"}`)
		dispatch(t)
		neverShownDispatched := getTestNotification(t, s.ds, neverShown)
		require.NotNil(t, neverShownDispatched.ExecutionID)
		_, neverShownToken := fetchScript(t, host, *neverShownDispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", neverShownToken, neverShown),
			[]byte(`{"action": "delay"}`), http.StatusNoContent)

		got = getTestNotification(t, s.ds, neverShown)
		require.NotNil(t, got.NextAttemptAt)
		require.WithinDuration(t, time.Now().UTC().Add(notifications_api.EndUserNotificationDelayInterval), *got.NextAttemptAt, time.Minute)
	})

	t.Run("POST delay with no kind registered is a no-op", func(t *testing.T) {
		host := newNotifiableHost(t, "notif-delay-unregistered")
		notificationUUID := newTestNotification(t, s.ds, host.ID, "some_unregistered_kind", `{"title": "hello"}`)
		dispatch(t)
		dispatched := getTestNotification(t, s.ds, notificationUUID)
		require.NotNil(t, dispatched.ExecutionID)
		_, token := fetchScript(t, host, *dispatched.ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notificationUUID),
			[]byte(`{"action": "delay"}`), http.StatusNoContent)

		got := getTestNotification(t, s.ds, notificationUUID)
		require.Equal(t, notifications_api.EndUserNotificationDispatched, got.Status, "nothing should have applied the delay")
		require.NotNil(t, got.ExecutionID)
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
		require.NotNil(t, second.ExecutionID, "the second notification should dispatch once the first is no longer in flight")
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

		// once it runs, it leaves no trace in the past activity feed: Fleet ran
		// this for itself, so there's no admin action to report
		postScriptResult(host, *dispatched.ExecutionID, 0)

		var past listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &past)
		for _, act := range past.Activities {
			if act.Details != nil {
				require.NotContains(t, string(*act.Details), *dispatched.ExecutionID,
					"the notification's script should not appear in past activities")
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

		// a re-dispatch moves execution_id on, so nothing points at the script
		// already on the host
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
}
