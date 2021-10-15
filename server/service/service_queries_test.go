package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueryAttach(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	name := "bad"
	query := "attach '/nope' as bad"
	_, err := svc.NewQuery(
		context.Background(),
		fleet.QueryPayload{Name: &name, Query: &query},
	)
	require.Error(t, err)
}

func TestFilterQueriesForObserver(t *testing.T) {
	require.True(t, onlyShowObserverCanRunQueries(&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}))
	require.False(t, onlyShowObserverCanRunQueries(&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)}))
	require.False(t, onlyShowObserverCanRunQueries(&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}))

	require.True(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{{Role: fleet.RoleObserver}}}))
	require.True(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{
		{Role: fleet.RoleObserver},
		{Role: fleet.RoleObserver},
	}}))
	require.False(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{
		{Role: fleet.RoleObserver},
		{Role: fleet.RoleMaintainer},
	}}))
}

func TestListQueries(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	cases := [...]struct {
		title        string
		user         *fleet.User
		expectedOpts fleet.ListQueryOptions
	}{
		{
			title:        "global admin",
			user:         &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			expectedOpts: fleet.ListQueryOptions{OnlyObserverCanRun: false},
		},
		{
			title:        "global observer",
			user:         &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			expectedOpts: fleet.ListQueryOptions{OnlyObserverCanRun: true},
		},
		{
			title:        "team maintainer",
			user:         &fleet.User{Teams: []fleet.UserTeam{{Role: fleet.RoleMaintainer}}},
			expectedOpts: fleet.ListQueryOptions{OnlyObserverCanRun: false},
		},
	}

	var calledWithOpts fleet.ListQueryOptions
	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, error) {
		calledWithOpts = opt
		return []*fleet.Query{}, nil
	}

	for _, tt := range cases {
		t.Run(tt.title, func(t *testing.T) {
			viewerCtx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			_, err := svc.ListQueries(viewerCtx, fleet.ListOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOpts, calledWithOpts)
		})
	}
}

func TestQueryAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)
	authoredQueryID := uint(1)
	authoredQueryName := "authored"
	queryName := map[uint]string{
		authoredQueryID: authoredQueryName,
		2:               "not authored",
	}
	teamMaintainer := &fleet.User{ID: 42, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}

	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, name string, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		if name == authoredQueryName {
			return &fleet.Query{ID: 99, AuthorID: ptr.Uint(teamMaintainer.ID)}, nil
		}
		return &fleet.Query{ID: 8888, AuthorID: ptr.Uint(6666)}, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == authoredQueryID {
			return &fleet.Query{ID: 99, AuthorID: ptr.Uint(teamMaintainer.ID)}, nil
		}
		return &fleet.Query{ID: 8888, AuthorID: ptr.Uint(6666)}, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query) error {
		return nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, name string) error {
		return nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		return 0, nil
	}

	var testCases = []struct {
		name            string
		user            *fleet.User
		qid             uint
		shouldFailWrite bool
		shouldFailRead  bool
		shouldFailNew   bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			authoredQueryID,
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			authoredQueryID,
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			authoredQueryID,
			true,
			false,
			true,
		},
		{
			"team maintainer, author of the query",
			teamMaintainer,
			authoredQueryID,
			false,
			false,
			false,
		},
		{
			"team maintainer, NOT author of the query",
			teamMaintainer,
			2,
			true,
			false,
			false,
		},
		{
			"team observer",
			&fleet.User{ID: 48, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: authoredQueryID}, Role: fleet.RoleObserver}}},
			2,
			true,
			false,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.NewQuery(ctx, fleet.QueryPayload{Name: ptr.String("name"), Query: ptr.String("select 1")})
			checkAuthErr(t, tt.shouldFailNew, err)

			_, err = svc.ModifyQuery(ctx, tt.qid, fleet.QueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteQuery(ctx, queryName[tt.qid])
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteQueryByID(ctx, tt.qid)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteQueries(ctx, []uint{tt.qid})
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
