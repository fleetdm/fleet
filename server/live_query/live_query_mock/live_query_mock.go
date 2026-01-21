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
	GetQueryResultsCountOverride    func(queryID uint) (int, error)
	IncrQueryResultsCountOverride   func(queryID uint, amount int) (int, error)
	SetQueryResultsCountOverride    func(queryID uint, count int) error
	DeleteQueryResultsCountOverride func(queryID uint) error
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

// LoadActiveQueryNames mocks the live query store LoadActiveQueryNames method.
func (m *MockLiveQuery) LoadActiveQueryNames() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// GetQueryResultsCount mocks the live query store GetQueryResultsCount method.
func (m *MockLiveQuery) GetQueryResultsCount(queryID uint) (int, error) {
	if m.GetQueryResultsCountOverride != nil {
		return m.GetQueryResultsCountOverride(queryID)
	}
	args := m.Called(queryID)
	return args.Int(0), args.Error(1)
}

// IncrQueryResultsCount mocks the live query store IncrQueryResultsCount method.
func (m *MockLiveQuery) IncrQueryResultsCount(queryID uint, amount int) (int, error) {
	if m.IncrQueryResultsCountOverride != nil {
		return m.IncrQueryResultsCountOverride(queryID, amount)
	}
	args := m.Called(queryID, amount)
	return args.Int(0), args.Error(1)
}

// SetQueryResultsCount mocks the live query store SetQueryResultsCount method.
func (m *MockLiveQuery) SetQueryResultsCount(queryID uint, count int) error {
	if m.SetQueryResultsCountOverride != nil {
		return m.SetQueryResultsCountOverride(queryID, count)
	}
	args := m.Called(queryID, count)
	return args.Error(0)
}

// DeleteQueryResultsCount mocks the live query store DeleteQueryResultsCount method.
func (m *MockLiveQuery) DeleteQueryResultsCount(queryID uint) error {
	if m.DeleteQueryResultsCountOverride != nil {
		return m.DeleteQueryResultsCountOverride(queryID)
	}
	return nil
}
