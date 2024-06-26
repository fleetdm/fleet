package live_query_mock

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/mock"
)

// MockLiveQuery allows mocking a live query store.
type MockLiveQuery struct {
	mock.Mock
	fleet.LiveQueryStore
}

var _ fleet.LiveQueryStore = (*MockLiveQuery)(nil)

// New allocates a mocked live query store.
func New(t *testing.T) *MockLiveQuery {
	m := new(MockLiveQuery)
	m.Test(t)
	return m
}

// RunQuery mocks the live query store RunQuery method.
func (m *MockLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	args := m.Called(name, sql, hostIDs)
	return args.Error(0)
}

// StopQuery mocks the live query store StopQuery method.
func (m *MockLiveQuery) StopQuery(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// QueriesForHost mocks the live query store QueriesForHost method.
func (m *MockLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	args := m.Called(hostID)
	return args.Get(0).(map[string]string), args.Error(1)
}

// QueryCompletedByHost mocks the live query store QueryCompletedByHost method.
func (m *MockLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	args := m.Called(name, hostID)
	return args.Error(0)
}

// CleanupInactiveQueries mocks the live query store CleanupInactiveQueries method.
func (m *MockLiveQuery) CleanupInactiveQueries(ctx context.Context, inactiveCampaignIDs []uint) error {
	args := m.Called(ctx, inactiveCampaignIDs)
	return args.Error(0)
}
