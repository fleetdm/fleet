package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// dispatchTestNotifications mirrors cmd/fleet/cron.go's dispatchEndUserNotifications:
// list what's due, queue the script for each host, record the execution. The real
// cron lives in package main and can't be imported here, so this calls the same
// datastore methods in the same order.
func dispatchTestNotifications(t *testing.T, ds fleet.Datastore) []*fleet.EndUserNotification {
	t.Helper()
	ctx := context.Background()

	_, err := ds.ExpireEndUserNotifications(ctx)
	require.NoError(t, err)

	notifications, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
	require.NoError(t, err)
	if len(notifications) == 0 {
		return nil
	}

	hostIDs := make([]uint, 0, len(notifications))
	for _, n := range notifications {
		hostIDs = append(hostIDs, n.HostID)
	}
	executionIDByHost, err := ds.BatchNewInternalHostScriptExecutionRequests(ctx, hostIDs, fleet.EndUserNotificationScript)
	require.NoError(t, err)
	for _, n := range notifications {
		execID := executionIDByHost[n.HostID]
		n.ExecutionID = &execID
	}
	require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, notifications))
	return notifications
}

func (s *integrationTestSuite) TestEndUserNotifications() {
	t := s.T()
	ctx := context.Background()

	newNotification := func(t *testing.T, hostID uint, kind string, payload string) *fleet.EndUserNotification {
		created, err := s.ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
			HostID:  hostID,
			Kind:    kind,
			Payload: json.RawMessage(payload),
		})
		require.NoError(t, err)
		return created
	}

	// fetches the notification's script as orbit would, returning the substituted
	// script contents and the device token minted along the way
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
		host := createOrbitEnrolledHost(t, "darwin", "notif-dispatch", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)

		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		require.Equal(t, notification.UUID, dispatched[0].UUID)
		require.NotNil(t, dispatched[0].ExecutionID)

		scriptContents, token := fetchScript(t, host, *dispatched[0].ExecutionID)
		require.NotEmpty(t, token)
		require.Contains(t, scriptContents, fmt.Sprintf("/device/%s/notifications/%s", token, notification.UUID))
		require.NotContains(t, scriptContents, "FLEET_VAR")

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, fleet.EndUserNotificationDispatched, got.Status)
	})

	t.Run("exit 0 records displayed_at", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-exit0", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)

		postScriptResult(host, *dispatched[0].ExecutionID, 0)

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, fleet.EndUserNotificationDispatched, got.Status)
		require.NotNil(t, got.DisplayedAt)
		require.NotNil(t, got.LastExitCode)
		require.EqualValues(t, 0, *got.LastExitCode)
	})

	t.Run("exit 41 schedules a retry", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-exit41", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)

		postScriptResult(host, *dispatched[0].ExecutionID, 41)

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, fleet.EndUserNotificationPending, got.Status)
		require.Nil(t, got.DisplayedAt)
		require.NotNil(t, got.LastReason)
		require.Equal(t, fleet.EndUserNotificationReasonScreenLocked, *got.LastReason)
		require.NotNil(t, got.NextAttemptAt)
	})

	t.Run("exit 2 is a terminal failure", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-exit2", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)

		postScriptResult(host, *dispatched[0].ExecutionID, 2)

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, fleet.EndUserNotificationFailed, got.Status)
		require.Nil(t, got.NextAttemptAt)
	})

	t.Run("GET returns the notification's payload", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-get", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello world"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		var resp struct {
			Payload json.RawMessage `json:"payload"`
			Err     string          `json:"error,omitempty"`
		}
		s.DoJSONWithoutAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", token, notification.UUID),
			nil, http.StatusOK, &resp)
		require.JSONEq(t, `{"title": "hello world"}`, string(resp.Payload))
	})

	t.Run("GET with another host's token 404s", func(t *testing.T) {
		hostA := createOrbitEnrolledHost(t, "darwin", "notif-cross-a", s.ds)
		hostB := createOrbitEnrolledHost(t, "darwin", "notif-cross-b", s.ds)
		notification := newNotification(t, hostA.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatchedA := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatchedA, 1)
		_, tokenA := fetchScript(t, hostA, *dispatchedA[0].ExecutionID)
		require.NotEmpty(t, tokenA)

		// mint hostB its own token by fetching a script of its own
		otherNotification := newNotification(t, hostB.ID, "notify_before_patching", `{"title": "other"}`)
		dispatchedB := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatchedB, 1)
		require.Equal(t, otherNotification.UUID, dispatchedB[0].UUID)
		_, tokenB := fetchScript(t, hostB, *dispatchedB[0].ExecutionID)

		s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s", tokenB, notification.UUID),
			nil, http.StatusNotFound)
	})

	t.Run("GET with an unknown uuid 404s", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-unknown-uuid", s.ds)
		newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/no-such-uuid", token),
			nil, http.StatusNotFound)
	})

	t.Run("POST verify sets displayed_at, a second call doesn't move it", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-verify", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notification.UUID),
			[]byte(`{"action": "verify"}`), http.StatusNoContent)

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.NotNil(t, got.DisplayedAt)
		firstDisplayedAt := *got.DisplayedAt

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notification.UUID),
			fmt.Appendf(nil, `{"action": "verify", "displayed_at": %q}`, time.Now().Add(time.Minute).Format(time.RFC3339)),
			http.StatusNoContent)

		got, err = s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, firstDisplayedAt, *got.DisplayedAt)
	})

	t.Run("POST delay with a registered kind delays it", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-delay-registered", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notification.UUID),
			[]byte(`{"action": "delay"}`), http.StatusNoContent)

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, fleet.EndUserNotificationPending, got.Status)
		require.Nil(t, got.ExecutionID)
		require.NotNil(t, got.LastReason)
		require.Equal(t, fleet.EndUserNotificationReasonDelayed, *got.LastReason)
		require.NotNil(t, got.NextAttemptAt)
	})

	t.Run("POST delay with no kind registered is a no-op", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-delay-unregistered", s.ds)
		notification := newNotification(t, host.ID, "some_unregistered_kind", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notification.UUID),
			[]byte(`{"action": "delay"}`), http.StatusNoContent)

		got, err := s.ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		require.Equal(t, fleet.EndUserNotificationDispatched, got.Status, "nothing should have applied the delay")
		require.NotNil(t, got.ExecutionID)
	})

	t.Run("POST with an unknown action is rejected", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-bad-action", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notification.UUID),
			[]byte(`{"action": "dance"}`), http.StatusUnprocessableEntity)
	})

	t.Run("POST with a missing action is rejected", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-no-action", s.ds)
		notification := newNotification(t, host.ID, "notify_before_patching", `{"title": "hello"}`)
		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)

		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, notification.UUID),
			[]byte(`{}`), http.StatusUnprocessableEntity)
	})

	t.Run("a second due notification on the same host waits its turn", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "darwin", "notif-in-flight", s.ds)
		first := newNotification(t, host.ID, "notify_before_patching", `{"title": "first"}`)
		second := newNotification(t, host.ID, "notify_before_patching", `{"title": "second"}`)

		dispatched := dispatchTestNotifications(t, s.ds)
		require.Len(t, dispatched, 1)
		require.Equal(t, first.UUID, dispatched[0].UUID)

		stillWaiting := dispatchTestNotifications(t, s.ds)
		require.Empty(t, stillWaiting, "the host already has an undelivered dispatch")

		_, token := fetchScript(t, host, *dispatched[0].ExecutionID)
		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/notifications/%s/actions", token, first.UUID),
			[]byte(`{"action": "verify"}`), http.StatusNoContent)

		nowFree := dispatchTestNotifications(t, s.ds)
		require.Len(t, nowFree, 1)
		require.Equal(t, second.UUID, nowFree[0].UUID)
	})
}
