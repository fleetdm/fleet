package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// MockACMEService is a mock implementation of fleet.ACMEWriteService
// for unit tests that use mock.Store instead of real MySQL connections.
type MockACMEService struct {
	UpsertEnrollmentFunc        func(ctx context.Context, hostIdentifier string) (string, error)
	UpsertEnrollmentFuncInvoked bool
}

// Ensure MockACMEService implements fleet.ACMEWriteService.
var _ fleet.ACMEWriteService = (*MockACMEService)(nil)

func (m *MockACMEService) UpsertEnrollment(ctx context.Context, hostIdentifier string) (string, error) {
	m.UpsertEnrollmentFuncInvoked = true
	if m.UpsertEnrollmentFunc != nil {
		return m.UpsertEnrollmentFunc(ctx, hostIdentifier)
	}
	return "", nil
}
