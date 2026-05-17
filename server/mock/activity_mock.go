package mock

import (
	"context"
	"sync"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// NewActivityFunc is the callback function type for MockActivityService.
type NewActivityFunc func(ctx context.Context, user *activity_api.User, activity activity_api.ActivityDetails) error

// NoopNewActivityFunc is a no-op implementation of NewActivityFunc for tests
// that don't need to intercept activity creation.
var NoopNewActivityFunc NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
	return nil
}

// MockActivityService is a mock implementation of fleet.ActivityWriteService
// for unit tests that use mock.Store instead of real MySQL connections.
// When Delegate is set, it is called before the mock's NewActivityFunc,
// allowing real behavior (e.g. webhooks) while still capturing calls.
type MockActivityService struct {
	NewActivityFunc        NewActivityFunc // defaults to NoopNewActivityFunc if nil
	NewActivityFuncInvoked bool
	Delegate               activity_api.NewActivityService

	CleanupHostActivitiesFunc        func(ctx context.Context, hostIDs []uint) error
	CleanupHostActivitiesFuncInvoked bool

	ListHostPastActivitiesForDeviceFunc        func(ctx context.Context, hostID uint, opt activity_api.ListOptions) ([]*activity_api.Activity, *activity_api.PaginationMetadata, error)
	ListHostPastActivitiesForDeviceFuncInvoked bool

	mu sync.Mutex
}

// Ensure MockActivityService implements fleet.ActivityWriteService.
var _ fleet.ActivityWriteService = (*MockActivityService)(nil)

func (m *MockActivityService) NewActivity(ctx context.Context, user *activity_api.User, activity activity_api.ActivityDetails) error {
	m.mu.Lock()
	m.NewActivityFuncInvoked = true
	m.mu.Unlock()
	if m.Delegate != nil {
		if err := m.Delegate.NewActivity(ctx, user, activity); err != nil {
			return err
		}
	}
	fn := m.NewActivityFunc
	if fn == nil {
		fn = NoopNewActivityFunc
	}
	return fn(ctx, user, activity)
}

func (m *MockActivityService) CleanupHostActivities(ctx context.Context, hostIDs []uint) error {
	m.mu.Lock()
	m.CleanupHostActivitiesFuncInvoked = true
	m.mu.Unlock()
	if m.CleanupHostActivitiesFunc != nil {
		return m.CleanupHostActivitiesFunc(ctx, hostIDs)
	}
	return nil
}

func (m *MockActivityService) ListHostPastActivitiesForDevice(ctx context.Context, hostID uint, opt activity_api.ListOptions) ([]*activity_api.Activity, *activity_api.PaginationMetadata, error) {
	m.mu.Lock()
	m.ListHostPastActivitiesForDeviceFuncInvoked = true
	m.mu.Unlock()
	if m.ListHostPastActivitiesForDeviceFunc != nil {
		return m.ListHostPastActivitiesForDeviceFunc(ctx, hostID, opt)
	}
	return nil, &activity_api.PaginationMetadata{}, nil
}
