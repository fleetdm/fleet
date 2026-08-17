package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
	platform_errors "github.com/fleetdm/fleet/v4/server/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndUserNotifications(t *testing.T) {
	env := newTestEnv(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"NewAndGet", testNewAndGetEndUserNotification},
		{"ListToDispatch", testListEndUserNotificationsToDispatch},
		{"SetDispatched", testSetEndUserNotificationsDispatched},
		{"DeferForHosts", testDeferEndUserNotificationsForHosts},
		{"Expire", testExpireEndUserNotifications},
		{"Verify", testVerifyEndUserNotification},
		{"Delay", testDelayEndUserNotification},
		{"Outcome", testSetEndUserNotificationOutcome},
		{"HostDeleteCascade", testEndUserNotificationHostDeleteCascade},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

// newDarwinHost creates a darwin host with orbit info, so it's eligible for dispatch.
func newDarwinHost(t *testing.T, env *testEnv, name string, withOrbitInfo bool) uint {
	hostID := env.InsertHost(t, name, "darwin")
	if withOrbitInfo {
		env.InsertHostOrbitInfo(t, hostID)
	}
	return hostID
}

func testNewAndGetEndUserNotification(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := newDarwinHost(t, env, "new-and-get", true)

	created, err := env.ds.NewEndUserNotification(ctx, &api.EndUserNotification{
		HostID:  hostID,
		Kind:    "test_kind",
		Payload: []byte(`{"title": "hello world"}`),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.UUID)
	assert.Equal(t, api.EndUserNotificationPending, created.Status)
	assert.Equal(t, hostID, created.HostID)
	assert.Equal(t, "test_kind", created.Kind)
	assert.JSONEq(t, `{"title": "hello world"}`, string(created.Payload))
	assert.Nil(t, created.ExecutionID)
	assert.Nil(t, created.DisplayedAt)

	got, err := env.ds.GetEndUserNotificationByUUID(ctx, created.UUID)
	require.NoError(t, err)
	assert.Equal(t, created.UUID, got.UUID)

	_, err = env.ds.GetEndUserNotificationByUUID(ctx, "no-such-uuid")
	assert.True(t, platform_errors.IsNotFound(err))

	_, err = env.ds.GetEndUserNotificationByExecutionID(ctx, "no-such-execution-id")
	assert.True(t, platform_errors.IsNotFound(err))
}

func testListEndUserNotificationsToDispatch(t *testing.T, env *testEnv) {
	ctx := t.Context()

	t.Run("one per host, oldest first", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := newDarwinHost(t, env, "dispatch-one-per-host", true)
		olderUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		env.InsertNotification(t, hostID, "k", nil, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		require.Len(t, due, 1)
		assert.Equal(t, olderUUID, due[0].UUID)
	})

	t.Run("non-darwin host excluded", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := env.InsertHost(t, "dispatch-linux", "linux")
		env.InsertHostOrbitInfo(t, hostID)
		env.InsertNotification(t, hostID, "k", nil, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("host without orbit info excluded", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := newDarwinHost(t, env, "dispatch-no-orbit-info", false)
		env.InsertNotification(t, hostID, "k", nil, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("future next_attempt_at excluded", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := newDarwinHost(t, env, "dispatch-future", true)
		future := time.Now().Add(time.Hour)
		env.InsertNotification(t, hostID, "k", &future, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("expired excluded even before the sweep runs", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := newDarwinHost(t, env, "dispatch-expired", true)
		past := time.Now().Add(-time.Hour)
		env.InsertNotification(t, hostID, "k", nil, &past)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("host with an undelivered dispatch is skipped entirely", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := newDarwinHost(t, env, "dispatch-in-flight", true)
		inFlightUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, inFlightUUID, hostID)))
		env.InsertNotification(t, hostID, "k", nil, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("host with an already-displayed dispatch is not busy", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostID := newDarwinHost(t, env, "dispatch-already-displayed", true)
		displayedUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, displayedUUID, hostID)))
		require.NoError(t, env.ds.VerifyEndUserNotification(ctx, displayedUUID, time.Now()))
		nextUUID := env.InsertNotification(t, hostID, "k", nil, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		require.Len(t, due, 1)
		assert.Equal(t, nextUUID, due[0].UUID)
	})

	t.Run("different hosts go out together, and limit caps rows read", func(t *testing.T) {
		defer env.TruncateTables(t)
		hostA := newDarwinHost(t, env, "dispatch-two-hosts-a", true)
		hostB := newDarwinHost(t, env, "dispatch-two-hosts-b", true)
		firstUUID := env.InsertNotification(t, hostA, "k", nil, nil)
		secondUUID := env.InsertNotification(t, hostB, "k", nil, nil)

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		require.Len(t, due, 2, "one host in flight doesn't hold up another")
		assert.ElementsMatch(t, []string{firstUUID, secondUUID}, []string{due[0].UUID, due[1].UUID})

		due, err = env.ds.ListEndUserNotificationsToDispatch(ctx, 1)
		require.NoError(t, err)
		require.Len(t, due, 1)
	})
}

// withExecutionID is the argument the dispatcher builds after queueing a script
// for each notification.
func withExecutionID(t *testing.T, env *testEnv, notificationUUID string, hostID uint) []*api.EndUserNotification {
	executionID := notificationUUID + "-exec"
	return []*api.EndUserNotification{{UUID: notificationUUID, HostID: hostID, ExecutionID: &executionID}}
}

func testSetEndUserNotificationsDispatched(t *testing.T, env *testEnv) {
	ctx := t.Context()

	t.Run("empty is a no-op", func(t *testing.T) {
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, nil))
	})

	t.Run("without an execution id is an error", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "dispatched-no-exec-id", true)
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		err := env.ds.SetEndUserNotificationsDispatched(ctx, []*api.EndUserNotification{{UUID: notificationUUID, HostID: hostID}})
		assert.Error(t, err)
	})

	t.Run("records the execution and bumps attempt count", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "dispatched-ok", true)
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		notifications := withExecutionID(t, env, notificationUUID, hostID)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, notifications))

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationDispatched, got.Status)
		require.NotNil(t, got.ExecutionID)
		assert.Equal(t, *notifications[0].ExecutionID, *got.ExecutionID)
		assert.EqualValues(t, 1, got.AttemptCount)
	})

}

func testExpireEndUserNotifications(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := newDarwinHost(t, env, "expire", true)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	pastNoExpiryUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	pastExpiredUUID := env.InsertNotification(t, hostID, "k", nil, &past)
	pastButDisplayedUUID := env.InsertNotification(t, hostID, "k", nil, &past)
	require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, pastButDisplayedUUID, hostID)))
	require.NoError(t, env.ds.VerifyEndUserNotification(ctx, pastButDisplayedUUID, time.Now()))
	futureExpiryUUID := env.InsertNotification(t, hostID, "k", nil, &future)

	// dispatched, no expires_at, untouched for longer than the timeout: the
	// host never came back with a result
	stuckDispatchedUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, stuckDispatchedUUID, hostID)))
	longAgo := time.Now().Add(-api.EndUserNotificationStuckDispatchTimeout - time.Hour)
	_, err := env.db.ExecContext(ctx, "UPDATE notifications_end_user SET updated_at = ? WHERE uuid = ?", longAgo, stuckDispatchedUUID)
	require.NoError(t, err)

	// dispatched just now, no expires_at: still in flight, must be left alone
	recentlyDispatchedUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, recentlyDispatchedUUID, hostID)))

	count, err := env.ds.ExpireEndUserNotifications(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)

	assertStatus := func(uuid, want string) {
		got, err := env.ds.GetEndUserNotificationByUUID(ctx, uuid)
		require.NoError(t, err)
		assert.Equal(t, want, got.Status)
	}
	assertStatus(pastExpiredUUID, api.EndUserNotificationExpired)
	assertStatus(pastButDisplayedUUID, api.EndUserNotificationDispatched)
	assertStatus(futureExpiryUUID, api.EndUserNotificationPending)
	assertStatus(pastNoExpiryUUID, api.EndUserNotificationPending)
	assertStatus(stuckDispatchedUUID, api.EndUserNotificationExpired)
	assertStatus(recentlyDispatchedUUID, api.EndUserNotificationDispatched)
}

func testVerifyEndUserNotification(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := newDarwinHost(t, env, "verify", true)
	notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, notificationUUID, hostID)))

	first := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	require.NoError(t, env.ds.VerifyEndUserNotification(ctx, notificationUUID, first))

	got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
	require.NoError(t, err)
	require.NotNil(t, got.DisplayedAt)
	assert.WithinDuration(t, first, *got.DisplayedAt, time.Second)

	later := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, env.ds.VerifyEndUserNotification(ctx, notificationUUID, later))

	got, err = env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
	require.NoError(t, err)
	assert.WithinDuration(t, first, *got.DisplayedAt, time.Second)

	// a result that lands after the end user delayed it belongs to a send that is
	// over, so it can't mark the next one as displayed
	delayedUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, delayedUUID, hostID)))
	require.NoError(t, env.ds.DelayEndUserNotification(ctx, delayedUUID, time.Now().Add(time.Hour), nil))
	require.NoError(t, env.ds.VerifyEndUserNotification(ctx, delayedUUID, time.Now()))

	got, err = env.ds.GetEndUserNotificationByUUID(ctx, delayedUUID)
	require.NoError(t, err)
	assert.Nil(t, got.DisplayedAt)
}

func testDelayEndUserNotification(t *testing.T, env *testEnv) {
	ctx := t.Context()

	t.Run("delays a dispatched notification", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "delay", true)
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		notifications := withExecutionID(t, env, notificationUUID, hostID)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, notifications))

		nextAttempt := time.Now().Add(55 * time.Minute).UTC().Truncate(time.Second)
		require.NoError(t, env.ds.DelayEndUserNotification(ctx, notificationUUID, nextAttempt, nil))

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationPending, got.Status)
		require.NotNil(t, got.ExecutionID, "keeps its execution_id so a late result can still find it")
		assert.Equal(t, *notifications[0].ExecutionID, *got.ExecutionID)
		assert.Nil(t, got.DisplayedAt, "the next send records its own display")
		require.NotNil(t, got.LastReason)
		assert.Equal(t, api.EndUserNotificationReasonDelayed, *got.LastReason)
		require.NotNil(t, got.NextAttemptAt)
		assert.WithinDuration(t, nextAttempt, *got.NextAttemptAt, time.Second)
	})

	t.Run("new content replaces the payload under the same uuid", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "delay-reminder", true)
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, notificationUUID, hostID)))

		require.NoError(t, env.ds.DelayEndUserNotification(ctx, notificationUUID, time.Now().Add(time.Minute),
			[]byte(`{"title": "5 minutes left"}`)))

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, notificationUUID, got.UUID, "a reminder stays the same notification")
		assert.JSONEq(t, `{"title": "5 minutes left"}`, string(got.Payload))
	})

	t.Run("a re-dispatched notification still blocks the host", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "delay-inflight", true)
		first := env.InsertNotification(t, hostID, "k", nil, nil)
		second := env.InsertNotification(t, hostID, "k", nil, nil)

		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, first, hostID)))
		require.NoError(t, env.ds.VerifyEndUserNotification(ctx, first, time.Now()))
		require.NoError(t, env.ds.DelayEndUserNotification(ctx, first, time.Now().Add(-time.Minute), nil))

		redispatch := first + "-exec2"
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx,
			[]*api.EndUserNotification{{UUID: first, HostID: hostID, ExecutionID: &redispatch}}))

		due, err := env.ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		for _, n := range due {
			assert.NotEqual(t, second, n.UUID, "it went out while the re-dispatched one was still in flight")
		}
	})

	t.Run("does not resurrect an already-expired notification", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "delay-expired", true)
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, notificationUUID, hostID)))
		require.NoError(t, env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			ExitCode: 2, Reason: api.EndUserNotificationReasonBadInvocation,
		}, nil))
		// the outcome above leaves it failed, so set expired directly
		_, err := env.db.ExecContext(ctx, "UPDATE notifications_end_user SET status = ? WHERE uuid = ?",
			api.EndUserNotificationExpired, notificationUUID)
		require.NoError(t, err)

		require.NoError(t, env.ds.DelayEndUserNotification(ctx, notificationUUID, time.Now().Add(time.Hour), nil))

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationExpired, got.Status, "delay must not revive an expired notification")
	})

	t.Run("does not resurrect an already-failed notification", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "delay-failed", true)
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, notificationUUID, hostID)))
		require.NoError(t, env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			ExitCode: 2, Reason: api.EndUserNotificationReasonBadInvocation,
		}, nil))

		require.NoError(t, env.ds.DelayEndUserNotification(ctx, notificationUUID, time.Now().Add(time.Hour), nil))

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationFailed, got.Status, "delay must not revive a failed notification")
	})
}

func testSetEndUserNotificationOutcome(t *testing.T, env *testEnv) {
	ctx := t.Context()

	dispatch := func(t *testing.T, hostID uint) string {
		notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)
		require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, notificationUUID, hostID)))
		return notificationUUID
	}

	t.Run("displayed delegates to VerifyEndUserNotification", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "outcome-displayed", true)
		notificationUUID := dispatch(t, hostID)

		err := env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			Displayed: true, ExitCode: 0,
		}, nil)
		require.NoError(t, err)

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationDispatched, got.Status)
		require.NotNil(t, got.DisplayedAt)
		require.NotNil(t, got.LastExitCode)
		assert.EqualValues(t, 0, *got.LastExitCode)
	})

	t.Run("failure with a retry goes back to pending", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "outcome-retry", true)
		notificationUUID := dispatch(t, hostID)
		nextAttempt := time.Now().Add(30 * time.Second)

		err := env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			ExitCode: 41, Reason: api.EndUserNotificationReasonScreenLocked,
		}, &nextAttempt)
		require.NoError(t, err)

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationPending, got.Status)
		require.NotNil(t, got.NextAttemptAt)
		assert.WithinDuration(t, nextAttempt, *got.NextAttemptAt, time.Second)
		require.NotNil(t, got.LastReason)
		assert.Equal(t, api.EndUserNotificationReasonScreenLocked, *got.LastReason)
		assert.Nil(t, got.DisplayedAt)
	})

	t.Run("terminal failure with no retry", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "outcome-terminal", true)
		notificationUUID := dispatch(t, hostID)

		err := env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			ExitCode: 2, Reason: api.EndUserNotificationReasonBadInvocation,
		}, nil)
		require.NoError(t, err)

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationFailed, got.Status)
		assert.Nil(t, got.NextAttemptAt)
	})

	t.Run("a late result can't revive an expired notification", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "outcome-expired", true)
		notificationUUID := dispatch(t, hostID)
		past := time.Now().Add(-time.Hour)
		_, err := env.db.ExecContext(ctx, "UPDATE notifications_end_user SET expires_at = ? WHERE uuid = ?", past, notificationUUID)
		require.NoError(t, err)

		nextAttempt := time.Now().Add(30 * time.Second)
		err = env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			ExitCode: 41, Reason: api.EndUserNotificationReasonScreenLocked,
		}, &nextAttempt)
		require.NoError(t, err)

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.Equal(t, api.EndUserNotificationDispatched, got.Status, "status must not move once past expires_at")
		assert.Nil(t, got.NextAttemptAt)
		require.NotNil(t, got.LastReason, "the reason is still recorded even though the transition is skipped")
		assert.Equal(t, api.EndUserNotificationReasonScreenLocked, *got.LastReason)
	})

	t.Run("a genuine display still counts even past expires_at", func(t *testing.T) {
		hostID := newDarwinHost(t, env, "outcome-expired-displayed", true)
		notificationUUID := dispatch(t, hostID)
		past := time.Now().Add(-time.Hour)
		_, err := env.db.ExecContext(ctx, "UPDATE notifications_end_user SET expires_at = ? WHERE uuid = ?", past, notificationUUID)
		require.NoError(t, err)

		err = env.ds.SetEndUserNotificationOutcome(ctx, notificationUUID, api.NotificationOutcome{
			Displayed: true, ExitCode: 0,
		}, nil)
		require.NoError(t, err)

		got, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
		require.NoError(t, err)
		assert.NotNil(t, got.DisplayedAt)
	})
}

func testEndUserNotificationHostDeleteCascade(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := newDarwinHost(t, env, "cascade", true)
	notificationUUID := env.InsertNotification(t, hostID, "k", nil, nil)

	env.DeleteHost(t, hostID)

	_, err := env.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
	assert.True(t, platform_errors.IsNotFound(err))
}

func testDeferEndUserNotificationsForHosts(t *testing.T, env *testEnv) {
	ctx := t.Context()

	hostID := newDarwinHost(t, env, "defer", true)
	sentUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	waitingUUID := env.InsertNotification(t, hostID, "k", nil, nil)
	otherHostID := newDarwinHost(t, env, "defer-other-host", true)
	untouchedUUID := env.InsertNotification(t, otherHostID, "k", nil, nil)

	require.NoError(t, env.ds.SetEndUserNotificationsDispatched(ctx, withExecutionID(t, env, sentUUID, hostID)))
	require.NoError(t, env.ds.DeferEndUserNotificationsForHosts(ctx, []uint{hostID}))

	waiting, err := env.ds.GetEndUserNotificationByUUID(ctx, waitingUUID)
	require.NoError(t, err)
	assert.Equal(t, api.EndUserNotificationPending, waiting.Status, "it waits its turn rather than failing")
	require.NotNil(t, waiting.LastReason)
	assert.Equal(t, api.EndUserNotificationReasonDeferred, *waiting.LastReason)

	sent, err := env.ds.GetEndUserNotificationByUUID(ctx, sentUUID)
	require.NoError(t, err)
	assert.Nil(t, sent.LastReason, "the one that went out doesn't defer itself")

	untouched, err := env.ds.GetEndUserNotificationByUUID(ctx, untouchedUUID)
	require.NoError(t, err)
	assert.Nil(t, untouched.LastReason, "another host's queue is unaffected")
}
