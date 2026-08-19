package mock

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type RecordOutcomeFunc func(ctx context.Context, executionID string, exitCode int64, output string) error

type NotificationUUIDForExecutionFunc func(ctx context.Context, executionID string) (string, error)

var NoopRecordOutcomeFunc RecordOutcomeFunc = func(_ context.Context, _ string, _ int64, _ string) error {
	return nil
}

// MockNotificationsService stands in for the notifications bounded context in
// tests that run on mock.Store rather than MySQL.
type MockNotificationsService struct {
	RecordOutcomeFunc        RecordOutcomeFunc // defaults to NoopRecordOutcomeFunc if nil
	RecordOutcomeFuncInvoked bool

	NotificationUUIDForExecutionFunc        NotificationUUIDForExecutionFunc
	NotificationUUIDForExecutionFuncInvoked bool

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

type notFoundError struct{}

func (e *notFoundError) Error() string    { return "notification not found" }
func (e *notFoundError) IsNotFound() bool { return true }
