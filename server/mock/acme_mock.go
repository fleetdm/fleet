package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// MockACMEService is a mock implementation of fleet.ACMEWriteService
// for unit tests that use mock.Store instead of real MySQL connections.
type MockACMEService struct {
	NewEnrollmentFunc        func(ctx context.Context, hostIdentifier string) (string, error)
	NewEnrollmentFuncInvoked bool
}

// Ensure MockACMEService implements fleet.ACMEWriteService.
var _ fleet.ACMEWriteService = (*MockACMEService)(nil)

func (m *MockACMEService) NewEnrollment(ctx context.Context, hostIdentifier string) (string, error) {
	m.NewEnrollmentFuncInvoked = true
	if m.NewEnrollmentFunc != nil {
		return m.NewEnrollmentFunc(ctx, hostIdentifier)
	}
	return "", nil
}
