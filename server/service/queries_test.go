package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryPayloadValidationCreate(t *testing.T) {
	ds := new(mock.Store)
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
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
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		assert.NotEmpty(t, query)
		return nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
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

	team2 := fleet.Team{
		ID:   2,
		Name: "Barfoo",
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
	team2Query := fleet.Query{
		ID:     77,
		Name:   "team2 query",
		TeamID: ptr.Uint(team2.ID),
	}
	queriesMap := map[uint]fleet.Query{
		globalQuery.ID: globalQuery,
		teamQuery.ID:   teamQuery,
		team2Query.ID:  team2Query,
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return &team, nil
		} else if tid == team2.ID {
			return &team2, nil
		}
		return nil, newNotFoundError()
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == team.Name {
			return &team, nil
		} else if name == team2.Name {
			return &team2, nil
		}
		return nil, newNotFoundError()
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		if teamID == nil && name == "global query" { //nolint:gocritic // ignore ifElseChain
			return &globalQuery, nil
		} else if teamID != nil && *teamID == team.ID && name == "team query" {
			return &teamQuery, nil
		} else if teamID != nil && *teamID == team2.ID && name == "team2 query" {
			return &team2Query, nil
		}
		return nil, newNotFoundError()
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == 99 { //nolint:gocritic // ignore ifElseChain
			return &globalQuery, nil
		} else if id == 88 {
			return &teamQuery, nil
		} else if id == 77 {
			return &team2Query, nil
		}
		return nil, newNotFoundError()
	}

	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}

	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		return nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		return 0, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.ApplyQueriesFunc = func(ctx context.Context, authID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
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
			false,
			false,
		},
		{
			"global gitops and team query",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			teamQuery.ID,
			false,
			false,
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
			"team admin and team2 query",
			teamAdmin,
			team2Query.ID,
			true,
			true,
			true,
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
			"team maintainer and team2 query",
			teamMaintainer,
			team2Query.ID,
			true,
			true,
			true,
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
			"team observer and team2 query",
			teamObserver,
			team2Query.ID,
			true,
			true,
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
			"team observer+ and team2 query",
			teamObserverPlus,
			team2Query.ID,
			true,
			true,
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
			false,
			false,
		},
		{
			"team gitops and team2 query",
			teamGitOps,
			team2Query.ID,
			true,
			true,
			true,
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

			_, err = svc.QueryReportIsClipped(ctx, tt.qid, fleet.DefaultMaxQueryReportRows)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, _, _, err = svc.ListQueries(ctx, fleet.ListOptions{}, query.TeamID, nil, false, nil)
			checkAuthErr(t, tt.shouldFailRead, err)

			teamName := ""
			if query.TeamID != nil && *query.TeamID == team.ID {
				teamName = team.Name
			} else if query.TeamID != nil && *query.TeamID == team2.ID {
				teamName = team2.Name
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

func TestQueryReportIsClipped(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})

	ds.QueryFunc = func(ctx context.Context, queryID uint) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}
	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}

	isClipped, err := svc.QueryReportIsClipped(viewerCtx, 1, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.False(t, isClipped)

	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return fleet.DefaultMaxQueryReportRows, nil
	}

	isClipped, err = svc.QueryReportIsClipped(viewerCtx, 1, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.True(t, isClipped)
}

func TestQueryReportReturnsNilIfDiscardDataIsTrue(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})

	ds.QueryFunc = func(ctx context.Context, queryID uint) (*fleet.Query, error) {
		return &fleet.Query{
			DiscardData: true,
		}, nil
	}
	ds.QueryResultRowsFunc = func(ctx context.Context, queryID uint, opts fleet.TeamFilter) ([]*fleet.ScheduledQueryResultRow, error) {
		return []*fleet.ScheduledQueryResultRow{
			{
				QueryID:     1,
				HostID:      1,
				Data:        ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
				LastFetched: time.Now(),
			},
		}, nil
	}

	results, reportClipped, err := svc.GetQueryReportResults(viewerCtx, 1)
	require.NoError(t, err)
	require.Nil(t, results)
	require.False(t, reportClipped)
}

func TestComparePlatforms(t *testing.T) {
	for _, tc := range []struct {
		name     string
		p1       string
		p2       string
		expected bool
	}{
		{
			name:     "equal single value",
			p1:       "linux",
			p2:       "linux",
			expected: true,
		},
		{
			name:     "different single value",
			p1:       "macos",
			p2:       "linux",
			expected: false,
		},
		{
			name:     "equal multiple values",
			p1:       "linux,windows",
			p2:       "linux,windows",
			expected: true,
		},
		{
			name:     "equal multiple values out of order",
			p1:       "linux,windows",
			p2:       "windows,linux",
			expected: true,
		},
		{
			name:     "different multiple values",
			p1:       "linux,windows",
			p2:       "linux,windows,darwin",
			expected: false,
		},
		{
			name:     "no values set",
			p1:       "",
			p2:       "",
			expected: true,
		},
		{
			name:     "no values set",
			p1:       "",
			p2:       "linux",
			expected: false,
		},
		{
			name:     "single and multiple values",
			p1:       "linux",
			p2:       "windows,linux",
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := comparePlatforms(tc.p1, tc.p2)
			require.Equal(t, tc.expected, actual)
		})
	}
}
