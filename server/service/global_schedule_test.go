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

func TestGlobalScheduleAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	//
	// All global schedule query methods use queries datastore methods.
	//

	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{
			Name:  "foobar",
			Query: "SELECT 1;",
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
	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
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
			shouldFailRead:  false,
		},
		{
			name:            "team admin",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
		{
			name:            "team maintainer",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
		{
			name:            "team observer",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetGlobalScheduledQueries(ctx, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GlobalScheduleQuery(ctx, &fleet.ScheduledQuery{
				Name:      "query",
				QueryName: "query",
				Interval:  10,
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ModifyGlobalScheduledQueries(ctx, 1, fleet.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteGlobalScheduledQueries(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
