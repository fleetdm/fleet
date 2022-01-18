package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func TestGlobalScheduleAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		return nil, nil
	}
	ds.EnsureGlobalPackFunc = func(ctx context.Context) (*fleet.Pack, error) {
		return &fleet.Pack{}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
		return sq, nil
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
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
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
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.GetGlobalScheduledQueries(ctx, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GlobalScheduleQuery(ctx, &fleet.ScheduledQuery{Name: "query", QueryName: "query"})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ModifyGlobalScheduledQueries(ctx, 1, fleet.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteGlobalScheduledQueries(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
