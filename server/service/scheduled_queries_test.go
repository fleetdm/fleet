package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledQueriesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		return nil, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		return sq, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}
	ds.ScheduledQueryFunc = func(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
		return &fleet.ScheduledQuery{}, nil
	}
	ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
		return sq, nil
	}
	ds.DeleteScheduledQueryFunc = func(ctx context.Context, id uint) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			name:            "global admin",
			user:            &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailWrite: false,
			shouldFailRead:  false,
		},
		{
			name:            "global maintainer",
			user:            &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailWrite: false,
			shouldFailRead:  false,
		},
		{
			name:            "global observer",
			user:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "global observer+",
			user:            &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "global gitops",
			user:            &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			shouldFailWrite: false,
			shouldFailRead:  false, // Global gitops can read packs (exception to the write only rule)
		},
		// Team users cannot read or write scheduled queries using the below service APIs.
		// Team users must use the "Team" endpoints (GetTeamScheduledQueries, TeamScheduleQuery,
		// ModifyTeamScheduledQueries and DeleteTeamScheduledQueries).
		{
			name:            "team admin",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team maintainer",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team observer",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team observer+",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team gitops",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetScheduledQueriesInPack(ctx, 1, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ScheduleQuery(ctx, &fleet.ScheduledQuery{Interval: 10})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetScheduledQuery(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyScheduledQuery(ctx, 1, fleet.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteScheduledQuery(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestScheduleQuery(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.NewScheduledQueryFunc = func(ctx context.Context, q *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(test.UserContext(ctx, test.UserAdmin), expectedQuery)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoName(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*fleet.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &fleet.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
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
		test.UserContext(ctx, test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 10},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoNameMultiple(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar-1",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*fleet.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &fleet.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		// No matching query
		return []*fleet.ScheduledQuery{
			{
				Name:     "foobar",
				Interval: 10,
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, q *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 10},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestFindNextNameForQuery(t *testing.T) {
	testCases := []struct {
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
		},
		{
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

func TestScheduleQueryInterval(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &fleet.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*fleet.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &fleet.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
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
		test.UserContext(ctx, test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 10},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)

	// no interval
	_, err = svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID},
	)
	assert.Error(t, err)

	// interval zero
	_, err = svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 0},
	)
	assert.Error(t, err)

	// interval exceeds max
	_, err = svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&fleet.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 604801},
	)
	assert.Error(t, err)
}

func TestModifyScheduledQueryInterval(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ScheduledQueryFunc = func(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
		assert.Equal(t, id, uint(1))
		return &fleet.ScheduledQuery{ID: id, Interval: 10}, nil
	}

	testCases := []struct {
		payload    fleet.ScheduledQueryPayload
		shouldFail bool
	}{
		{
			payload: fleet.ScheduledQueryPayload{
				QueryID:  ptr.Uint(1),
				Interval: ptr.Uint(0),
			},
			shouldFail: true,
		},
		{
			payload: fleet.ScheduledQueryPayload{
				QueryID:  ptr.Uint(1),
				Interval: ptr.Uint(604801),
			},
			shouldFail: true,
		},
		{
			payload: fleet.ScheduledQueryPayload{
				QueryID: ptr.Uint(1),
			},
			shouldFail: false,
		},
		{
			payload: fleet.ScheduledQueryPayload{
				QueryID:  ptr.Uint(1),
				Interval: ptr.Uint(604800),
			},
			shouldFail: false,
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
				assert.Equal(t, sq.ID, uint(1))
				return &fleet.ScheduledQuery{ID: sq.ID, Interval: sq.Interval}, nil
			}
			_, err := svc.ModifyScheduledQuery(test.UserContext(ctx, test.UserAdmin), *tt.payload.QueryID, tt.payload)
			if tt.shouldFail {
				assert.Error(t, err)
				assert.False(t, ds.SaveScheduledQueryFuncInvoked)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
