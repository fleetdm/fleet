package mysql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndUserNotifications(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewAndGet", testNewAndGetEndUserNotification},
		{"ListToDispatch", testListEndUserNotificationsToDispatch},
		{"SetDispatched", testSetEndUserNotificationsDispatched},
		{"Expire", testExpireEndUserNotifications},
		{"Verify", testVerifyEndUserNotification},
		{"Delay", testDelayEndUserNotification},
		{"Outcome", testSetEndUserNotificationOutcome},
		{"HostDeleteCascade", testEndUserNotificationHostDeleteCascade},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds, "end_user_notifications")
			c.fn(t, ds)
		})
	}
}

func newTestDarwinHost(t *testing.T, ds *Datastore, name string, withOrbitInfo bool) *fleet.Host {
	host := test.NewHost(t, ds, name, "1.1.1.1", name, name, time.Now())
	if withOrbitInfo {
		require.NoError(t, ds.SetOrUpdateHostOrbitInfo(
			context.Background(), host.ID, "1.40.0", sql.NullString{String: "1.5.0", Valid: true}, sql.NullBool{Bool: true, Valid: true},
		))
	}
	return host
}

func newTestNotification(t *testing.T, ds *Datastore, hostID uint, kind string) *fleet.EndUserNotification {
	created, err := ds.NewEndUserNotification(context.Background(), &fleet.EndUserNotification{
		HostID:  hostID,
		Kind:    kind,
		Payload: json.RawMessage(`{"title": "hello"}`),
	})
	require.NoError(t, err)
	return created
}

func testNewAndGetEndUserNotification(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newTestDarwinHost(t, ds, "new-and-get", true)

	created, err := ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
		HostID:  host.ID,
		Kind:    "test_kind",
		Payload: json.RawMessage(`{"title": "hello world"}`),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.UUID)
	assert.Equal(t, fleet.EndUserNotificationPending, created.Status)
	assert.Equal(t, host.ID, created.HostID)
	assert.Equal(t, "test_kind", created.Kind)
	assert.JSONEq(t, `{"title": "hello world"}`, string(created.Payload))
	assert.Nil(t, created.ExecutionID)
	assert.Nil(t, created.DisplayedAt)

	got, err := ds.GetEndUserNotificationByUUID(ctx, created.UUID)
	require.NoError(t, err)
	assert.Equal(t, created.UUID, got.UUID)

	_, err = ds.GetEndUserNotificationByUUID(ctx, "no-such-uuid")
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetEndUserNotificationByExecutionID(ctx, "no-such-execution-id")
	assert.True(t, fleet.IsNotFound(err))
}

func testListEndUserNotificationsToDispatch(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	t.Run("one per host, oldest first", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := newTestDarwinHost(t, ds, "dispatch-one-per-host", true)
		older := newTestNotification(t, ds, host.ID, "k")
		newTestNotification(t, ds, host.ID, "k")

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		require.Len(t, due, 1)
		assert.Equal(t, older.UUID, due[0].UUID)
	})

	t.Run("non-darwin host excluded", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := test.NewHost(t, ds, "dispatch-linux", "1.1.1.2", "dispatch-linux", "dispatch-linux", time.Now(), func(h *fleet.Host) { h.Platform = "linux" })
		require.NoError(t, ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, "1.40.0", sql.NullString{String: "1.5.0", Valid: true}, sql.NullBool{Bool: true, Valid: true}))
		newTestNotification(t, ds, host.ID, "k")

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("host without orbit info excluded", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := newTestDarwinHost(t, ds, "dispatch-no-orbit-info", false)
		newTestNotification(t, ds, host.ID, "k")

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("future next_attempt_at excluded", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := newTestDarwinHost(t, ds, "dispatch-future", true)
		future := time.Now().Add(time.Hour)
		_, err := ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
			HostID: host.ID, Kind: "k", Payload: json.RawMessage(`{}`), NextAttemptAt: &future,
		})
		require.NoError(t, err)

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("expired excluded even before the sweep runs", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := newTestDarwinHost(t, ds, "dispatch-expired", true)
		past := time.Now().Add(-time.Hour)
		_, err := ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
			HostID: host.ID, Kind: "k", Payload: json.RawMessage(`{}`), ExpiresAt: &past,
		})
		require.NoError(t, err)

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("host with an undelivered dispatch is skipped entirely", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := newTestDarwinHost(t, ds, "dispatch-in-flight", true)
		inFlight := newTestNotification(t, ds, host.ID, "k")
		require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, []*fleet.EndUserNotification{
			withExecutionID(t, ds, inFlight, host.ID),
		}))
		newTestNotification(t, ds, host.ID, "k")

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("host with an already-displayed dispatch is not busy", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		host := newTestDarwinHost(t, ds, "dispatch-already-displayed", true)
		displayed := newTestNotification(t, ds, host.ID, "k")
		require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, []*fleet.EndUserNotification{
			withExecutionID(t, ds, displayed, host.ID),
		}))
		require.NoError(t, ds.VerifyEndUserNotification(ctx, displayed.UUID, time.Now()))
		next := newTestNotification(t, ds, host.ID, "k")

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 500)
		require.NoError(t, err)
		require.Len(t, due, 1)
		assert.Equal(t, next.UUID, due[0].UUID)
	})

	t.Run("limit bounds notifications read, dedup still applies", func(t *testing.T) {
		defer TruncateTables(t, ds, "end_user_notifications")
		hostA := newTestDarwinHost(t, ds, "dispatch-limit-a", true)
		hostB := newTestDarwinHost(t, ds, "dispatch-limit-b", true)
		newTestNotification(t, ds, hostA.ID, "k")
		newTestNotification(t, ds, hostB.ID, "k")

		due, err := ds.ListEndUserNotificationsToDispatch(ctx, 1)
		require.NoError(t, err)
		require.Len(t, due, 1)
	})
}

// withExecutionID gives a notification a real execution to dispatch as, so
// SetEndUserNotificationsDispatched's join to upcoming_activities has something
// to match.
func withExecutionID(t *testing.T, ds *Datastore, notification *fleet.EndUserNotification, hostID uint) *fleet.EndUserNotification {
	queued, err := ds.NewInternalHostScriptExecutionRequest(context.Background(), &fleet.HostScriptRequestPayload{
		HostID:         hostID,
		ScriptContents: "echo hi",
	})
	require.NoError(t, err)
	notification.ExecutionID = &queued.ExecutionID
	return notification
}

func testSetEndUserNotificationsDispatched(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	t.Run("empty is a no-op", func(t *testing.T) {
		require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, nil))
	})

	t.Run("without an execution id is an error", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "dispatched-no-exec-id", true)
		notification := newTestNotification(t, ds, host.ID, "k")
		err := ds.SetEndUserNotificationsDispatched(ctx, []*fleet.EndUserNotification{notification})
		assert.Error(t, err)
	})

	t.Run("records the execution and bumps attempt count", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "dispatched-ok", true)
		notification := newTestNotification(t, ds, host.ID, "k")
		require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, []*fleet.EndUserNotification{
			withExecutionID(t, ds, notification, host.ID),
		}))

		got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		assert.Equal(t, fleet.EndUserNotificationDispatched, got.Status)
		require.NotNil(t, got.ExecutionID)
		assert.Equal(t, *notification.ExecutionID, *got.ExecutionID)
		assert.EqualValues(t, 1, got.AttemptCount)
	})
}

func testExpireEndUserNotifications(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newTestDarwinHost(t, ds, "expire", true)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	pastNoExpiry := newTestNotification(t, ds, host.ID, "k")

	pastExpired, err := ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
		HostID: host.ID, Kind: "k", Payload: json.RawMessage(`{}`), ExpiresAt: &past,
	})
	require.NoError(t, err)

	pastButDisplayed, err := ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
		HostID: host.ID, Kind: "k", Payload: json.RawMessage(`{}`), ExpiresAt: &past,
	})
	require.NoError(t, err)
	require.NoError(t, ds.VerifyEndUserNotification(ctx, pastButDisplayed.UUID, time.Now()))

	futureExpiry, err := ds.NewEndUserNotification(ctx, &fleet.EndUserNotification{
		HostID: host.ID, Kind: "k", Payload: json.RawMessage(`{}`), ExpiresAt: &future,
	})
	require.NoError(t, err)

	count, err := ds.ExpireEndUserNotifications(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)

	assertStatus := func(uuid, want string) {
		got, err := ds.GetEndUserNotificationByUUID(ctx, uuid)
		require.NoError(t, err)
		assert.Equal(t, want, got.Status)
	}
	assertStatus(pastExpired.UUID, fleet.EndUserNotificationExpired)
	assertStatus(pastButDisplayed.UUID, fleet.EndUserNotificationPending)
	assertStatus(futureExpiry.UUID, fleet.EndUserNotificationPending)
	assertStatus(pastNoExpiry.UUID, fleet.EndUserNotificationPending)
}

func testVerifyEndUserNotification(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newTestDarwinHost(t, ds, "verify", true)
	notification := newTestNotification(t, ds, host.ID, "k")

	first := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	require.NoError(t, ds.VerifyEndUserNotification(ctx, notification.UUID, first))

	got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
	require.NoError(t, err)
	require.NotNil(t, got.DisplayedAt)
	assert.WithinDuration(t, first, *got.DisplayedAt, time.Second)

	later := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, ds.VerifyEndUserNotification(ctx, notification.UUID, later))

	got, err = ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
	require.NoError(t, err)
	assert.WithinDuration(t, first, *got.DisplayedAt, time.Second)
}

func testDelayEndUserNotification(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newTestDarwinHost(t, ds, "delay", true)
	notification := newTestNotification(t, ds, host.ID, "k")
	require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, []*fleet.EndUserNotification{
		withExecutionID(t, ds, notification, host.ID),
	}))

	nextAttempt := time.Now().Add(55 * time.Minute).UTC().Truncate(time.Second)
	require.NoError(t, ds.DelayEndUserNotification(ctx, notification.UUID, nextAttempt))

	got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
	require.NoError(t, err)
	assert.Equal(t, fleet.EndUserNotificationPending, got.Status)
	assert.Nil(t, got.ExecutionID)
	require.NotNil(t, got.LastReason)
	assert.Equal(t, fleet.EndUserNotificationReasonDelayed, *got.LastReason)
	require.NotNil(t, got.NextAttemptAt)
	assert.WithinDuration(t, nextAttempt, *got.NextAttemptAt, time.Second)
}

func testSetEndUserNotificationOutcome(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	dispatch := func(t *testing.T, host *fleet.Host) *fleet.EndUserNotification {
		notification := newTestNotification(t, ds, host.ID, "k")
		require.NoError(t, ds.SetEndUserNotificationsDispatched(ctx, []*fleet.EndUserNotification{
			withExecutionID(t, ds, notification, host.ID),
		}))
		return notification
	}

	t.Run("displayed delegates to VerifyEndUserNotification", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "outcome-displayed", true)
		notification := dispatch(t, host)

		err := ds.SetEndUserNotificationOutcome(ctx, notification.UUID, fleet.NotificationOutcome{
			Displayed: true, ExitCode: 0,
		}, nil)
		require.NoError(t, err)

		got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		assert.Equal(t, fleet.EndUserNotificationDispatched, got.Status)
		require.NotNil(t, got.DisplayedAt)
		require.NotNil(t, got.LastExitCode)
		assert.EqualValues(t, 0, *got.LastExitCode)
	})

	t.Run("failure with a retry goes back to pending", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "outcome-retry", true)
		notification := dispatch(t, host)
		nextAttempt := time.Now().Add(30 * time.Second)

		err := ds.SetEndUserNotificationOutcome(ctx, notification.UUID, fleet.NotificationOutcome{
			ExitCode: 41, Reason: fleet.EndUserNotificationReasonScreenLocked,
		}, &nextAttempt)
		require.NoError(t, err)

		got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		assert.Equal(t, fleet.EndUserNotificationPending, got.Status)
		require.NotNil(t, got.NextAttemptAt)
		assert.WithinDuration(t, nextAttempt, *got.NextAttemptAt, time.Second)
		require.NotNil(t, got.LastReason)
		assert.Equal(t, fleet.EndUserNotificationReasonScreenLocked, *got.LastReason)
		assert.Nil(t, got.DisplayedAt)
	})

	t.Run("terminal failure with no retry", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "outcome-terminal", true)
		notification := dispatch(t, host)

		err := ds.SetEndUserNotificationOutcome(ctx, notification.UUID, fleet.NotificationOutcome{
			ExitCode: 2, Reason: fleet.EndUserNotificationReasonBadInvocation,
		}, nil)
		require.NoError(t, err)

		got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		assert.Equal(t, fleet.EndUserNotificationFailed, got.Status)
		assert.Nil(t, got.NextAttemptAt)
	})

	t.Run("a late result can't revive an expired notification", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "outcome-expired", true)
		notification := dispatch(t, host)
		past := time.Now().Add(-time.Hour)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE end_user_notifications SET expires_at = ? WHERE uuid = ?", past, notification.UUID)
			return err
		})

		nextAttempt := time.Now().Add(30 * time.Second)
		err := ds.SetEndUserNotificationOutcome(ctx, notification.UUID, fleet.NotificationOutcome{
			ExitCode: 41, Reason: fleet.EndUserNotificationReasonScreenLocked,
		}, &nextAttempt)
		require.NoError(t, err)

		got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		assert.Equal(t, fleet.EndUserNotificationDispatched, got.Status, "status must not move once past expires_at")
		assert.Nil(t, got.NextAttemptAt)
		require.NotNil(t, got.LastReason, "the reason is still recorded even though the transition is skipped")
		assert.Equal(t, fleet.EndUserNotificationReasonScreenLocked, *got.LastReason)
	})

	t.Run("a genuine display still counts even past expires_at", func(t *testing.T) {
		host := newTestDarwinHost(t, ds, "outcome-expired-displayed", true)
		notification := dispatch(t, host)
		past := time.Now().Add(-time.Hour)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE end_user_notifications SET expires_at = ? WHERE uuid = ?", past, notification.UUID)
			return err
		})

		err := ds.SetEndUserNotificationOutcome(ctx, notification.UUID, fleet.NotificationOutcome{
			Displayed: true, ExitCode: 0,
		}, nil)
		require.NoError(t, err)

		got, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
		require.NoError(t, err)
		assert.NotNil(t, got.DisplayedAt)
	})
}

func testEndUserNotificationHostDeleteCascade(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newTestDarwinHost(t, ds, "cascade", true)
	notification := newTestNotification(t, ds, host.ID, "k")

	require.NoError(t, ds.DeleteHost(ctx, host.ID))

	_, err := ds.GetEndUserNotificationByUUID(ctx, notification.UUID)
	assert.True(t, fleet.IsNotFound(err))
}
