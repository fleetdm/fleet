package live_query_mock

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/mock"
)

// MockLiveQuery allows mocking a live query store.
type MockLiveQuery struct {
	mock.Mock
	fleet.LiveQueryStore
	GetQueryResultsCountsOverride            func(queryIDs []uint) (map[uint]int, error)
	IncrQueryResultsCountsOverride           func(queryIDsToAmounts map[uint]int) error
	SetQueryResultsCountOverride             func(queryID uint, count int) error
	DeleteQueryResultsCountOverride          func(queryID uint) error
	SetQueryReportsHostCountOverride         func(count int) error
	GetQueryReportsHostCountOverride         func() (int, bool, error)
	SetQueryReportsHostCountIfAbsentOverride func(count int) error
	SetQueryResultsCountsIfAbsentOverride    func(counts map[uint]int) error
	IncrQueryReportsHostCountOverride        func(delta int) error
	MarkQueryReportsClippedOverride          func(ttlByQueryID map[uint]time.Duration) error
	QueryReportsClippedOverride              func(queryIDs []uint) (map[uint]bool, error)
	ClearQueryReportsClippedOverride         func(queryIDs []uint) error
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
func (m *MockLiveQuery) QueryCompletedByHost(name string, hostID uint) (bool, error) {
	args := m.Called(name, hostID)
	return args.Bool(0), args.Error(1)
}

// RestoreQueryTargetForHost mocks the live query store RestoreQueryTargetForHost method.
func (m *MockLiveQuery) RestoreQueryTargetForHost(name string, hostID uint) error {
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

// GetQueryResultsCounts mocks the live query store GetQueryResultsCounts method.
func (m *MockLiveQuery) GetQueryResultsCounts(queryIDs []uint) (map[uint]int, error) {
	if m.GetQueryResultsCountsOverride != nil {
		return m.GetQueryResultsCountsOverride(queryIDs)
	}
	args := m.Called(queryIDs)
	return args.Get(0).(map[uint]int), args.Error(1)
}

// IncrQueryResultsCounts mocks the live query store IncrQueryResultsCounts method.
func (m *MockLiveQuery) IncrQueryResultsCounts(queryIDsToAmounts map[uint]int) error {
	if m.IncrQueryResultsCountsOverride != nil {
		return m.IncrQueryResultsCountsOverride(queryIDsToAmounts)
	}
	args := m.Called(queryIDsToAmounts)
	return args.Error(0)
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

// SetQueryReportsHostCount mocks the live query store SetQueryReportsHostCount method.
func (m *MockLiveQuery) SetQueryReportsHostCount(count int) error {
	if m.SetQueryReportsHostCountOverride != nil {
		return m.SetQueryReportsHostCountOverride(count)
	}
	return nil
}

// GetQueryReportsHostCount mocks the live query store GetQueryReportsHostCount method.
func (m *MockLiveQuery) GetQueryReportsHostCount() (int, bool, error) {
	if m.GetQueryReportsHostCountOverride != nil {
		return m.GetQueryReportsHostCountOverride()
	}
	// Default to a cache hit of zero so tests opt into the database fallback explicitly.
	return 0, true, nil
}

// SetQueryReportsHostCountIfAbsent mocks the live query store SetQueryReportsHostCountIfAbsent method.
func (m *MockLiveQuery) SetQueryReportsHostCountIfAbsent(count int) error {
	if m.SetQueryReportsHostCountIfAbsentOverride != nil {
		return m.SetQueryReportsHostCountIfAbsentOverride(count)
	}
	return nil
}

// SetQueryResultsCountsIfAbsent mocks the live query store SetQueryResultsCountsIfAbsent method.
func (m *MockLiveQuery) SetQueryResultsCountsIfAbsent(counts map[uint]int) error {
	if m.SetQueryResultsCountsIfAbsentOverride != nil {
		return m.SetQueryResultsCountsIfAbsentOverride(counts)
	}
	return nil
}

// IncrQueryReportsHostCount mocks the live query store IncrQueryReportsHostCount method.
func (m *MockLiveQuery) IncrQueryReportsHostCount(delta int) error {
	if m.IncrQueryReportsHostCountOverride != nil {
		return m.IncrQueryReportsHostCountOverride(delta)
	}
	return nil
}

// MarkQueryReportsClipped mocks the live query store MarkQueryReportsClipped method.
func (m *MockLiveQuery) MarkQueryReportsClipped(ttlByQueryID map[uint]time.Duration) error {
	if m.MarkQueryReportsClippedOverride != nil {
		return m.MarkQueryReportsClippedOverride(ttlByQueryID)
	}
	return nil
}

// QueryReportsClipped mocks the live query store QueryReportsClipped method.
func (m *MockLiveQuery) QueryReportsClipped(queryIDs []uint) (map[uint]bool, error) {
	if m.QueryReportsClippedOverride != nil {
		return m.QueryReportsClippedOverride(queryIDs)
	}
	return map[uint]bool{}, nil
}

// ClearQueryReportsClipped mocks the live query store ClearQueryReportsClipped method.
func (m *MockLiveQuery) ClearQueryReportsClipped(queryIDs []uint) error {
	if m.ClearQueryReportsClippedOverride != nil {
		return m.ClearQueryReportsClippedOverride(queryIDs)
	}
	return nil
}
