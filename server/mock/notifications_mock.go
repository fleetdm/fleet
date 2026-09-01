package mock

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/google/uuid"
)

type RecordOutcomeFunc func(ctx context.Context, executionID string, exitCode int64, output string) error

type NotificationUUIDForExecutionFunc func(ctx context.Context, executionID string) (string, error)

type CreateNotificationFunc func(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.EndUserNotification, error)

type NotificationAwaitingFirstDispatchFunc func(ctx context.Context, hostID uint, kind string) (*notifications_api.EndUserNotification, error)

var NoopRecordOutcomeFunc RecordOutcomeFunc = func(_ context.Context, _ string, _ int64, _ string) error {
	return nil
}

var NoopCreateNotificationFunc CreateNotificationFunc = func(_ context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.EndUserNotification, error) {
	created := *notification
	if created.UUID == "" {
		created.UUID = uuid.NewString()
	}
	return &created, nil
}

// MockNotificationsService stands in for the notifications bounded context in
// tests that run on mock.Store rather than MySQL.
type MockNotificationsService struct {
	RecordOutcomeFunc        RecordOutcomeFunc // defaults to NoopRecordOutcomeFunc if nil
	RecordOutcomeFuncInvoked bool

	NotificationUUIDForExecutionFunc        NotificationUUIDForExecutionFunc
	NotificationUUIDForExecutionFuncInvoked bool

	CreateNotificationFunc        CreateNotificationFunc
	CreateNotificationFuncInvoked bool

	NotificationAwaitingFirstDispatchFunc        NotificationAwaitingFirstDispatchFunc
	NotificationAwaitingFirstDispatchFuncInvoked bool

	mu sync.Mutex
}

var _ fleet.NotificationsWriteService = (*MockNotificationsService)(nil)

func (m *MockNotificationsService) RecordOutcome(ctx context.Context, executionID string, exitCode int64, output string) error {
	m.mu.Lock()
	m.RecordOutcomeFuncInvoked = true
	m.mu.Unlock()
	fn := m.RecordOutcomeFunc
	if fn == nil {
		fn = NoopRecordOutcomeFunc
	}
	return fn(ctx, executionID, exitCode, output)
}

func (m *MockNotificationsService) NotificationUUIDForExecution(ctx context.Context, executionID string) (string, error) {
	m.mu.Lock()
	m.NotificationUUIDForExecutionFuncInvoked = true
	m.mu.Unlock()
	if m.NotificationUUIDForExecutionFunc == nil {
		return "", &notFoundError{}
	}
	return m.NotificationUUIDForExecutionFunc(ctx, executionID)
}

func (m *MockNotificationsService) CreateNotification(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.EndUserNotification, error) {
	m.mu.Lock()
	m.CreateNotificationFuncInvoked = true
	m.mu.Unlock()
	fn := m.CreateNotificationFunc
	if fn == nil {
		fn = NoopCreateNotificationFunc
	}
	return fn(ctx, notification)
}

func (m *MockNotificationsService) NotificationAwaitingFirstDispatch(ctx context.Context, hostID uint, kind string) (*notifications_api.EndUserNotification, error) {
	m.mu.Lock()
	m.NotificationAwaitingFirstDispatchFuncInvoked = true
	m.mu.Unlock()
	if m.NotificationAwaitingFirstDispatchFunc == nil {
		return nil, nil
	}
	return m.NotificationAwaitingFirstDispatchFunc(ctx, hostID, kind)
}

type notFoundError struct{}

func (e *notFoundError) Error() string    { return "notification not found" }
func (e *notFoundError) IsNotFound() bool { return true }
