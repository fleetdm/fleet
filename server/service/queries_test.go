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

func TestFilterQueriesForObserver(t *testing.T) {
	t.Run("global role", func(t *testing.T) {
		require.True(t, onlyShowObserverCanRunQueries(&fleet.User{
			GlobalRole: ptr.String(fleet.RoleObserver),
		}, nil))

		require.False(t, onlyShowObserverCanRunQueries(&fleet.User{
			GlobalRole: ptr.String(fleet.RoleObserverPlus),
		}, nil))

		require.False(t, onlyShowObserverCanRunQueries(&fleet.User{
			GlobalRole: ptr.String(fleet.RoleMaintainer),
		}, nil))

		require.False(t, onlyShowObserverCanRunQueries(&fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}, nil))
	})

	t.Run("user belongs to one or more teams", func(t *testing.T) {
		require.True(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{{
			Role: fleet.RoleObserver,
			Team: fleet.Team{ID: 1},
		}}}, ptr.Uint(1)))

		require.True(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{
			{
				Role: fleet.RoleObserver,
				Team: fleet.Team{ID: 1},
			},
			{
				Role: fleet.RoleObserver,
				Team: fleet.Team{ID: 2},
			},
		}}, ptr.Uint(2)))

		require.True(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{
			{
				Role: fleet.RoleObserver,
				Team: fleet.Team{ID: 1},
			},
			{
				Role: fleet.RoleMaintainer,
				Team: fleet.Team{ID: 2},
			},
		}}, ptr.Uint(1)))

		require.False(t, onlyShowObserverCanRunQueries(&fleet.User{Teams: []fleet.UserTeam{
			{
				Role: fleet.RoleObserver,
				Team: fleet.Team{ID: 1},
			},
			{
				Role: fleet.RoleMaintainer,
				Team: fleet.Team{ID: 2},
			},
		}}, ptr.Uint(2)))
	})
}

func TestListQueries(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

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
			viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			_, err := svc.ListQueries(viewerCtx, fleet.ListOptions{}, nil, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOpts, calledWithOpts)
		})
	}
}

func TestQueryPayloadValidationCreate(t *testing.T) {
	ds := new(mock.Store)
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		act, ok := activity.(fleet.ActivityTypeCreatedSavedQuery)
		assert.True(t, ok)
		assert.NotEmpty(t, act.Name)
		return nil
	}
	svc, ctx := newTestService(t, ds, nil, nil)

	testCases := []struct {
		name         string
		queryPayload fleet.QueryPayload
		shouldErr    bool
	}{
		{
			"All valid",
			fleet.QueryPayload{
				Name:     ptr.String("test query"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			false,
		},
		{
			"Invalid  - empty string name",
			fleet.QueryPayload{
				Name:     ptr.String(""),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Empty SQL",
			fleet.QueryPayload{
				Name:     ptr.String("bad sql"),
				Query:    ptr.String(""),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Invalid logging",
			fleet.QueryPayload{
				Name:     ptr.String("bad logging"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("hopscotch"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Unsupported platform",
			fleet.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("charles"),
			},
			true,
		},
		{
			"Missing comma",
			fleet.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin windows"),
			},
			true,
		},
		{
			"Unsupported platform 'sphinx' ",
			fleet.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin,windows,sphinx"),
			},
			true,
		},
	}

	testAdmin := fleet.User{
		ID:         1,
		Teams:      []fleet.UserTeam{},
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})
			query, err := svc.NewQuery(viewerCtx, tt.queryPayload)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Nil(t, query)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, query)
			}
		})
	}
}

// similar for modify
func TestQueryPayloadValidationModify(t *testing.T) {
	ds := new(mock.Store)
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{
			ID:             id,
			Name:           "mock saved query",
			Description:    "some desc",
			Query:          "select 1;",
			Platform:       "",
			Saved:          true,
			ObserverCanRun: false,
		}, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool) error {
		assert.NotEmpty(t, query)
		return nil
	}

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		act, ok := activity.(fleet.ActivityTypeEditedSavedQuery)
		assert.True(t, ok)
		assert.NotEmpty(t, act.Name)
		return nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	testCases := []struct {
		name         string
		queryPayload fleet.QueryPayload
		shouldErr    bool
	}{
		{
			"All valid",
			fleet.QueryPayload{
				Name:     ptr.String("updated test query"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			false,
		},
		{
			"Invalid  - empty string name",
			fleet.QueryPayload{
				Name:     ptr.String(""),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Empty SQL",
			fleet.QueryPayload{
				Name:     ptr.String("bad sql"),
				Query:    ptr.String(""),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Invalid logging",
			fleet.QueryPayload{
				Name:     ptr.String("bad logging"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("hopscotch"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Unsupported platform",
			fleet.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("charles"),
			},
			true,
		},
		{
			"Missing comma delimeter in platform string",
			fleet.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin windows"),
			},
			true,
		},
		{
			"Unsupported platform 2",
			fleet.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin,windows,sphinx"),
			},
			true,
		},
	}

	testAdmin := fleet.User{
		ID:         1,
		Teams:      []fleet.UserTeam{},
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})
			_, err := svc.ModifyQuery(viewerCtx, 1, tt.queryPayload)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQueryAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	team := fleet.Team{
		ID:   1,
		Name: "Foobar",
	}
	teamAdmin := &fleet.User{
		ID: 42,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: team.ID},
				Role: fleet.RoleAdmin,
			},
		},
	}
	teamMaintainer := &fleet.User{
		ID: 43,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: team.ID},
				Role: fleet.RoleMaintainer,
			},
		},
	}
	teamObserver := &fleet.User{
		ID: 44,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: team.ID},
				Role: fleet.RoleObserver,
			},
		},
	}
	teamObserverPlus := &fleet.User{
		ID: 45,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: team.ID},
				Role: fleet.RoleObserverPlus,
			},
		},
	}
	teamGitOps := &fleet.User{
		ID: 46,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: team.ID},
				Role: fleet.RoleGitOps,
			},
		},
	}
	globalQuery := fleet.Query{
		ID:     99,
		Name:   "global query",
		TeamID: nil,
	}
	teamQuery := fleet.Query{
		ID:     88,
		Name:   "team query",
		TeamID: ptr.Uint(team.ID),
	}
	queriesMap := map[uint]fleet.Query{
		globalQuery.ID: globalQuery,
		teamQuery.ID:   teamQuery,
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &team, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == team.Name {
			return &team, nil
		}
		return nil, newNotFoundError()
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		if teamID == nil && name == "global query" {
			return &globalQuery, nil
		} else if teamID != nil && *teamID == team.ID && name == "team query" {
			return &teamQuery, nil
		}
		return nil, newNotFoundError()
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == 99 {
			return &globalQuery, nil
		} else if id == 88 {
			return &teamQuery, nil
		}
		return nil, newNotFoundError()
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool) error {
		return nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		return 0, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) {
		return nil, nil
	}
	ds.ApplyQueriesFunc = func(ctx context.Context, authID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]bool) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		qid             uint
		shouldFailWrite bool
		shouldFailRead  bool
		shouldFailNew   bool
	}{
		{
			"global admin and global query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			globalQuery.ID,
			false,
			false,
			false,
		},
		{
			"global admin and team query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"global maintainer and global query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			globalQuery.ID,
			false,
			false,
			false,
		},
		{
			"global maintainer and team query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"global observer and global query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"global observer and team query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"global observer+ and global query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"global observer+ and team query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"global gitops and global query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			globalQuery.ID,
			false,
			true,
			false,
		},
		{
			"global gitops and team query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			teamQuery.ID,
			false,
			true,
			false,
		},
		{
			"team admin and global query",
			teamAdmin,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team admin and team query",
			teamAdmin,
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"team maintainer and global query",
			teamMaintainer,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team maintainer and team query",
			teamMaintainer,
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"team observer and global query",
			teamObserver,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer and team query",
			teamObserver,
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer+ and global query",
			teamObserverPlus,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer+ and team query",
			teamObserverPlus,
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"team gitops and global query",
			teamGitOps,
			globalQuery.ID,
			true,
			true,
			true,
		},
		{
			"team gitops and team query",
			teamGitOps,
			teamQuery.ID,
			false,
			true,
			false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			query := queriesMap[tt.qid]

			_, err := svc.NewQuery(ctx, fleet.QueryPayload{
				Name:   ptr.String("name"),
				Query:  ptr.String("select 1"),
				TeamID: query.TeamID,
			})
			checkAuthErr(t, tt.shouldFailNew, err)

			_, err = svc.ModifyQuery(ctx, tt.qid, fleet.QueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteQuery(ctx, query.TeamID, query.Name)
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteQueryByID(ctx, tt.qid)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteQueries(ctx, []uint{tt.qid})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetQuery(ctx, tt.qid)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ListQueries(ctx, fleet.ListOptions{}, query.TeamID, nil)
			checkAuthErr(t, tt.shouldFailRead, err)

			teamName := ""
			if query.TeamID != nil {
				teamName = team.Name
			}
			err = svc.ApplyQuerySpecs(ctx, []*fleet.QuerySpec{{
				Name:     query.Name,
				Query:    "SELECT 1",
				TeamName: teamName,
			}})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetQuerySpecs(ctx, query.TeamID)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetQuerySpec(ctx, query.TeamID, query.Name)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}
