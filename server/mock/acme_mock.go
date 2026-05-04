package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// MockACMEService is a mock implementation of fleet.ACMEWriteService
// for unit tests that use mock.Store instead of real MySQL connections.
type MockACMEService struct {
	NewACMEEnrollmentFunc        func(ctx context.Context, hostIdentifier string) (string, error)
	NewACMEEnrollmentFuncInvoked bool
}

// Ensure MockACMEService implements fleet.ACMEWriteService.
var _ fleet.ACMEWriteService = (*MockACMEService)(nil)

func (m *MockACMEService) NewACMEEnrollment(ctx context.Context, hostIdentifier string) (string, error) {
	m.NewACMEEnrollmentFuncInvoked = true
	if m.NewACMEEnrollmentFunc != nil {
		return m.NewACMEEnrollmentFunc(ctx, hostIdentifier)
	}
	return "", nil
}
