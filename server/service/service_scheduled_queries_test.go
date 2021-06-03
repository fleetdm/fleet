package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleQuery(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedQuery := &kolide.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
	}

	ds.NewScheduledQueryFunc = func(q *kolide.ScheduledQuery, opts ...kolide.OptionalArg) (*kolide.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(context.Background(), expectedQuery)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoName(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedQuery := &kolide.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
	}

	ds.QueryFunc = func(qid uint) (*kolide.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &kolide.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
		// No matching query
		return []*kolide.ScheduledQuery{
			&kolide.ScheduledQuery{
				Name: "froobling",
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(q *kolide.ScheduledQuery, opts ...kolide.OptionalArg) (*kolide.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		context.Background(),
		&kolide.ScheduledQuery{QueryID: expectedQuery.QueryID},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoNameMultiple(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedQuery := &kolide.ScheduledQuery{
		Name:      "foobar-1",
		QueryName: "foobar",
		QueryID:   3,
	}

	ds.QueryFunc = func(qid uint) (*kolide.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &kolide.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
		// No matching query
		return []*kolide.ScheduledQuery{
			&kolide.ScheduledQuery{
				Name: "foobar",
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(q *kolide.ScheduledQuery, opts ...kolide.OptionalArg) (*kolide.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		context.Background(),
		&kolide.ScheduledQuery{QueryID: expectedQuery.QueryID},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestFindNextNameForQuery(t *testing.T) {
	var testCases = []struct {
		name      string
		scheduled []*kolide.ScheduledQuery
		expected  string
	}{
		{
			name:      "foobar",
			scheduled: []*kolide.ScheduledQuery{},
			expected:  "foobar",
		},
		{
			name: "foobar",
			scheduled: []*kolide.ScheduledQuery{
				{
					Name: "foobar",
				},
			},
			expected: "foobar-1",
		}, {
			name: "foobar",
			scheduled: []*kolide.ScheduledQuery{
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
