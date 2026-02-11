package mock

import (
	"context"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
)

// NewActivityFunc is the callback function type for MockNewActivityService.
type NewActivityFunc func(ctx context.Context, user *activity_api.User, activity activity_api.ActivityDetails) error

// NoopNewActivityFunc is a no-op implementation of NewActivityFunc for tests
// that don't need to intercept activity creation.
var NoopNewActivityFunc NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
	return nil
}

// MockNewActivityService is a mock implementation of activity_api.NewActivityService
// for unit tests that use mock.Store instead of real MySQL connections.
// When Delegate is set, it is called before the mock's NewActivityFunc,
// allowing real behavior (e.g. webhooks) while still capturing calls.
type MockNewActivityService struct {
	NewActivityFunc        NewActivityFunc
	NewActivityFuncInvoked bool
	Delegate               activity_api.NewActivityService
}

// Ensure MockNewActivityService implements activity_api.NewActivityService.
var _ activity_api.NewActivityService = (*MockNewActivityService)(nil)

func (m *MockNewActivityService) NewActivity(ctx context.Context, user *activity_api.User, activity activity_api.ActivityDetails) error {
	m.NewActivityFuncInvoked = true
	if m.Delegate != nil {
		if err := m.Delegate.NewActivity(ctx, user, activity); err != nil {
			return err
		}
	}
	return m.NewActivityFunc(ctx, user, activity)
}
