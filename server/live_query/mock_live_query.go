package live_query

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/mock"
)

type MockLiveQuery struct {
	mock.Mock
	fleet.LiveQueryStore
}

func (m *MockLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	args := m.Called(name, sql, hostIDs)
	return args.Error(0)
}

func (m *MockLiveQuery) StopQuery(name string) error {
	args := m.Called(name)
	return args.Error(0)

}

func (m *MockLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	args := m.Called(hostID)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	args := m.Called(name, hostID)
	return args.Error(0)
}
