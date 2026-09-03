package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/notifications"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bare-bones mocks for testing the dispatcher in isolation.

type mockDatastore struct {
	due        []*api.EndUserNotification
	dispatched []*api.EndUserNotification
	deferred   []uint
}

func (m *mockDatastore) ExpireEndUserNotifications(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockDatastore) ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*api.EndUserNotification, error) {
	due := m.due
	m.due = nil
	return due, nil
}

func (m *mockDatastore) SetEndUserNotificationsDispatched(ctx context.Context, notifications []*api.EndUserNotification) error {
	m.dispatched = append(m.dispatched, notifications...)
	return nil
}

func (m *mockDatastore) DeferEndUserNotificationsForHosts(ctx context.Context, hostIDs []uint) error {
	m.deferred = append(m.deferred, hostIDs...)
	return nil
}

func (m *mockDatastore) GetEndUserNotificationByUUID(context.Context, string) (*api.EndUserNotification, error) {
	return nil, nil
}

func (m *mockDatastore) GetEndUserNotificationByExecutionID(context.Context, string) (*api.EndUserNotification, error) {
	return nil, nil
}

func (m *mockDatastore) VerifyEndUserNotification(context.Context, string, time.Time) error {
	return nil
}

func (m *mockDatastore) DelayEndUserNotification(context.Context, string, time.Time, json.RawMessage) error {
	return nil
}

func (m *mockDatastore) ActOnEndUserNotification(context.Context, string) (bool, error) {
	return true, nil
}

func (m *mockDatastore) SetEndUserNotificationOutcome(context.Context, string, api.NotificationOutcome, *time.Time) error {
	return nil
}

func (m *mockDatastore) NewEndUserNotification(context.Context, *api.EndUserNotification) (*api.EndUserNotification, error) {
	return nil, nil
}

func (m *mockDatastore) GetNotificationAwaitingDisplay(context.Context, uint, string) (*api.EndUserNotification, error) {
	return nil, nil
}

// mockScriptQueue returns the execution IDs it was built with, so a test can
// leave a host out.
type mockScriptQueue struct {
	executionIDByHost map[uint]string
}

func (m *mockScriptQueue) QueueScriptForHosts(ctx context.Context, hostIDs []uint, contents string) (map[uint]string, error) {
	return m.executionIDByHost, nil
}

var _ notifications.DataProviders = (*mockScriptQueue)(nil)

func TestExpireAndQueueNotifications(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	t.Run("marks each due notification with the script queued for its host", func(t *testing.T) {
		ds := &mockDatastore{due: []*api.EndUserNotification{
			{UUID: "notification-a", HostID: 1},
			{UUID: "notification-b", HostID: 2},
		}}
		queue := &mockScriptQueue{executionIDByHost: map[uint]string{1: "execution-a", 2: "execution-b"}}

		require.NoError(t, NewService(ds, queue, logger).ExpireAndQueueNotifications(ctx))

		require.Len(t, ds.dispatched, 2)
		require.NotNil(t, ds.dispatched[0].ExecutionID)
		assert.Equal(t, "execution-a", *ds.dispatched[0].ExecutionID)
		require.NotNil(t, ds.dispatched[1].ExecutionID)
		assert.Equal(t, "execution-b", *ds.dispatched[1].ExecutionID)
		assert.Equal(t, []uint{1, 2}, ds.deferred)
	})

	t.Run("a host the queue skipped fails the run rather than dispatching without a script", func(t *testing.T) {
		ds := &mockDatastore{due: []*api.EndUserNotification{
			{UUID: "notification-a", HostID: 1},
			{UUID: "notification-b", HostID: 2},
		}}
		queue := &mockScriptQueue{executionIDByHost: map[uint]string{1: "execution-a"}}

		err := NewService(ds, queue, logger).ExpireAndQueueNotifications(ctx)
		require.ErrorContains(t, err, "no script was queued for end user notification notification-b on host 2")

		// nothing is marked dispatched, so the next pass starts over rather than
		// leaving a notification pointing at a script that doesn't exist
		assert.Empty(t, ds.dispatched)
	})
}
