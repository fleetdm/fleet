package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func TestTeamScheduleAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == 99 { // for testing modify and delete of a schedule
			return &fleet.Query{
				Name:   "foobar",
				Query:  "SELECT 1;",
				TeamID: ptr.Uint(1),
			}, nil
		}
		return &fleet.Query{ // for testing creation of a schedule
			Name:  "foobar",
			Query: "SELECT 1;",
			// TeamID is set to nil because a query must be global to be able to be
			// scheduled on a team by the deprecated APIs.
			TeamID: nil,
		}, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			false, // global observer can view all queries and scheduled queries.
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
			false, // global observer+ can view all queries and scheduled queries.
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
			false,
		},
		{
			"team admin, belongs to team",
			&fleet.User{
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleAdmin,
				}},
			},
			false,
			false,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
			false,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			false,
		},
		{
			"team observer+, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
			false,
		},
		{
			"team gitops, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			false,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"team observer+, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserverPlus}}},
			true,
			true,
		},
		{
			"team gitops, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleGitOps}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetTeamScheduledQueries(ctx, 1, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.TeamScheduleQuery(ctx, 1, &fleet.ScheduledQuery{Interval: 10})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ModifyTeamScheduledQueries(ctx, 1, 99, fleet.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteTeamScheduledQueries(ctx, 1, 99)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
