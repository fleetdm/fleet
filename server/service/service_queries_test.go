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
			title:        "team admin",
			user:         &fleet.User{Teams: []fleet.UserTeam{{Role: fleet.RoleAdmin}}},
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
