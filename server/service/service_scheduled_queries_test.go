package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleQuery(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
	}

	ds.NewScheduledQueryFunc = func(ctx context.Context, q *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(test.UserContext(test.UserAdmin), expectedQuery)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoName(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*fleet.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &fleet.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		// No matching query
		return []*fleet.ScheduledQuery{
			{
				Name: "froobling",
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, q *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		test.UserContext(test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoNameMultiple(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar-1",
		QueryName: "foobar",
		QueryID:   3,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*fleet.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &fleet.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		// No matching query
		return []*fleet.ScheduledQuery{
			{
				Name: "foobar",
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, q *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		test.UserContext(test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestFindNextNameForQuery(t *testing.T) {
	var testCases = []struct {
		name      string
		scheduled []*fleet.ScheduledQuery
		expected  string
	}{
		{
			name:      "foobar",
			scheduled: []*fleet.ScheduledQuery{},
			expected:  "foobar",
		},
		{
			name: "foobar",
			scheduled: []*fleet.ScheduledQuery{
				{
					Name: "foobar",
				},
			},
			expected: "foobar-1",
		}, {
			name: "foobar",
			scheduled: []*fleet.ScheduledQuery{
				{
					Name: "foobar",
				},
				{
					Name: "foobar-1",
				},
			},
			expected: "foobar-1-1",
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.expected, findNextNameForQuery(tt.name, tt.scheduled))
		})
	}
}
