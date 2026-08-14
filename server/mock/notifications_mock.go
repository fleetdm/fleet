package mock

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// RecordOutcomeFunc is the callback function type for MockNotificationsService.
type RecordOutcomeFunc func(ctx context.Context, executionID string, exitCode int64, output string) error

// NotificationUUIDForExecutionFunc is the callback function type for MockNotificationsService.
type NotificationUUIDForExecutionFunc func(ctx context.Context, executionID string) (string, error)

// NoopRecordOutcomeFunc is a no-op implementation of RecordOutcomeFunc for
// tests that don't need to intercept notification outcome recording.
var NoopRecordOutcomeFunc RecordOutcomeFunc = func(_ context.Context, _ string, _ int64, _ string) error {
	return nil
}

// MockNotificationsService is a mock implementation of
// fleet.NotificationsWriteService for unit tests that use mock.Store instead
// of real MySQL connections.
type MockNotificationsService struct {
	RecordOutcomeFunc        RecordOutcomeFunc // defaults to NoopRecordOutcomeFunc if nil
	RecordOutcomeFuncInvoked bool

	NotificationUUIDForExecutionFunc        NotificationUUIDForExecutionFunc
	NotificationUUIDForExecutionFuncInvoked bool

	mu sync.Mutex
}

// Ensure MockNotificationsService implements fleet.NotificationsWriteService.
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
