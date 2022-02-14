package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
)

type nopLiveQuery struct{}

func (nopLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	return nil
}

func (nopLiveQuery) StopQuery(name string) error {
	return nil
}

func (nopLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	return map[string]string{}, nil
}

func (nopLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	return nil
}

func TestLiveQueryAuth(t *testing.T) {
	ds := new(mock.Store)
	qr := pubsub.NewInmemQueryResults()
	svc := newTestService(ds, qr, nopLiveQuery{})

	teamMaintainer := &fleet.User{ID: 42, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}
	query1ObsCanRun := &fleet.Query{
		ID:             1,
		AuthorID:       ptr.Uint(teamMaintainer.ID),
		Name:           "q1",
		Query:          "SELECT 1",
		ObserverCanRun: true,
	}
	query2ObsCannotRun := &fleet.Query{
		ID:             2,
		AuthorID:       ptr.Uint(teamMaintainer.ID),
		Name:           "q2",
		Query:          "SELECT 2",
		ObserverCanRun: false,
	}

	var lastCreatedQuery *fleet.Query
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		q := *query
		vw, ok := viewer.FromContext(ctx)
		q.ID = 123
		if ok {
			q.AuthorID = ptr.Uint(vw.User.ID)
		}
		lastCreatedQuery = &q
		return &q, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{LiveQueryDisabled: false}}, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filters fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return nil, nil
	}
	ds.HostIDsByNameFunc = func(ctx context.Context, filter fleet.TeamFilter, names []string) ([]uint, error) {
		return nil, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) ([]uint, error) {
		return nil, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filters fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == 1 {
			return query1ObsCanRun, nil
		}
		if id == 2 {
			return query2ObsCannotRun, nil
		}
		if lastCreatedQuery != nil {
			q := lastCreatedQuery
			lastCreatedQuery = nil
			return q, nil
		}
		return &fleet.Query{ID: 8888, AuthorID: ptr.Uint(6666)}, nil
	}

	testCases := []struct {
		name                   string
		user                   *fleet.User
		teamID                 *uint // to use as host target
		shouldFailRunNew       bool
		shouldFailRunObsCan    bool
		shouldFailRunObsCannot bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			nil,
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			nil,
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			nil,
			true,
			false,
			true,
		},
		{
			"team maintainer",
			teamMaintainer,
			nil,
			false,
			false,
			false,
		},
		{
			"team admin, target not set to own team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			ptr.Uint(2),
			false,
			true, // fails observer can run, as they are not part of that team, even as observer
			true,
		},
		{
			"team admin, target set to own team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			ptr.Uint(1),
			false,
			false,
			false,
		},
		{
			"team observer, target not set to own team",
			&fleet.User{ID: 48, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			ptr.Uint(2),
			true,
			true,
			true,
		},
		{
			"team observer, target set to own team",
			&fleet.User{ID: 48, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			ptr.Uint(1),
			true,
			false,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			var tms []uint
			// Testing RunNew is tricky, because RunNew authorization is done, then
			// the query is created, and then the Run authorization is applied to
			// that now-existing query, so we have to make sure that the Run does not
			// cause a Forbidden error. To this end, the ds.NewQuery mock always sets
			// the AuthorID to the context user, and if the user is member of a team,
			// always set that team as a host target. This will prevent the Run
			// action from failing, if RunNew did succeed.
			if len(tt.user.Teams) > 0 {
				tms = []uint{tt.user.Teams[0].ID}
			}
			_, err := svc.NewDistributedQueryCampaign(ctx, query1ObsCanRun.Query, nil, fleet.HostTargets{TeamIDs: tms})
			checkAuthErr(t, tt.shouldFailRunNew, err)

			if tt.teamID != nil {
				tms = []uint{*tt.teamID}
			}
			_, err = svc.NewDistributedQueryCampaign(ctx, query1ObsCanRun.Query, ptr.Uint(query1ObsCanRun.ID), fleet.HostTargets{TeamIDs: tms})
			checkAuthErr(t, tt.shouldFailRunObsCan, err)

			_, err = svc.NewDistributedQueryCampaign(ctx, query2ObsCannotRun.Query, ptr.Uint(query2ObsCannotRun.ID), fleet.HostTargets{TeamIDs: tms})
			checkAuthErr(t, tt.shouldFailRunObsCannot, err)

			// TODO: "by names" cannot work with team admin/maintainer/observer - it
			// does not support filter by team and it is required that they filter to
			// teams they admin/maintain/observe. Is it expected? Or should that be
			// allowed when no team is targeted?

			if len(tt.user.Teams) == 0 {
				_, err = svc.NewDistributedQueryCampaignByNames(ctx, query1ObsCanRun.Query, nil, nil, nil)
				checkAuthErr(t, tt.shouldFailRunNew, err)

				_, err = svc.NewDistributedQueryCampaignByNames(ctx, query1ObsCanRun.Query, ptr.Uint(query1ObsCanRun.ID), nil, nil)
				checkAuthErr(t, tt.shouldFailRunObsCan, err)

				_, err = svc.NewDistributedQueryCampaignByNames(ctx, query2ObsCannotRun.Query, ptr.Uint(query2ObsCannotRun.ID), nil, nil)
				checkAuthErr(t, tt.shouldFailRunObsCannot, err)
			}
		})
	}
}
