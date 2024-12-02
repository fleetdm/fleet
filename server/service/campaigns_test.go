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
	"github.com/stretchr/testify/require"
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

func (nopLiveQuery) CleanupInactiveQueries(ctx context.Context, inactiveCampaignIDs []uint) error {
	return nil
}

func (q nopLiveQuery) LoadActiveQueryNames() ([]string, error) {
	return nil, nil
}

func TestLiveQueryAuth(t *testing.T) {
	ds := new(mock.Store)
	qr := pubsub.NewInmemQueryResults()
	svc, ctx := newTestService(t, ds, qr, nopLiveQuery{})

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
		return []uint{1}, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, identifiers []string) ([]uint, error) {
		return nil, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) (map[string]uint, error) {
		return nil, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filters fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
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
			name:                   "global admin",
			user:                   &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			teamID:                 nil,
			shouldFailRunNew:       false,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: false,
		},
		{
			name:                   "global maintainer",
			user:                   &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			teamID:                 nil,
			shouldFailRunNew:       false,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: false,
		},
		{
			name:                   "global observer",
			user:                   &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			teamID:                 nil,
			shouldFailRunNew:       true,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: true,
		},
		{
			name:                   "team maintainer",
			user:                   teamMaintainer,
			teamID:                 nil,
			shouldFailRunNew:       false,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: false,
		},
		{
			name:                   "team admin, no team target",
			user:                   &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			teamID:                 nil,
			shouldFailRunNew:       false,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: false,
		},
		{
			name:                   "team admin, target not set to own team",
			user:                   &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			teamID:                 ptr.Uint(2),
			shouldFailRunNew:       false,
			shouldFailRunObsCan:    true, // fails observer can run, as they are not part of that team, even as observer
			shouldFailRunObsCannot: true,
		},
		{
			name:                   "team admin, target set to own team",
			user:                   &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			teamID:                 ptr.Uint(1),
			shouldFailRunNew:       false,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: false,
		},
		{
			name:                   "team observer, no team target",
			user:                   &fleet.User{ID: 48, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			teamID:                 nil,
			shouldFailRunNew:       true,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: true,
		},
		{
			name:                   "team observer, target not set to own team",
			user:                   &fleet.User{ID: 48, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			teamID:                 ptr.Uint(2),
			shouldFailRunNew:       true,
			shouldFailRunObsCan:    true,
			shouldFailRunObsCannot: true,
		},
		{
			name:                   "team observer, target set to own team",
			user:                   &fleet.User{ID: 48, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			teamID:                 ptr.Uint(1),
			shouldFailRunNew:       true,
			shouldFailRunObsCan:    false,
			shouldFailRunObsCannot: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

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

			// tests with a team target cannot run the "ByNames" calls, as there's no way
			// to pass a team target with this call.
			if tt.teamID == nil {
				_, err = svc.NewDistributedQueryCampaignByIdentifiers(ctx, query1ObsCanRun.Query, nil, nil, nil)
				checkAuthErr(t, tt.shouldFailRunNew, err)

				_, err = svc.NewDistributedQueryCampaignByIdentifiers(ctx, query1ObsCanRun.Query, ptr.Uint(query1ObsCanRun.ID), nil, nil)
				checkAuthErr(t, tt.shouldFailRunObsCan, err)

				_, err = svc.NewDistributedQueryCampaignByIdentifiers(ctx, query2ObsCannotRun.Query, ptr.Uint(query2ObsCannotRun.ID), nil, nil)
				checkAuthErr(t, tt.shouldFailRunObsCannot, err)
			}
		})
	}
}

func TestLiveQueryLabelValidation(t *testing.T) {
	ds := new(mock.Store)
	qr := pubsub.NewInmemQueryResults()
	svc, ctx := newTestService(t, ds, qr, nopLiveQuery{})

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	query := &fleet.Query{
		ID:             1,
		Name:           "q1",
		Query:          "SELECT 1",
		ObserverCanRun: true,
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		query.ID = 123
		return query, nil
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
		return []uint{1}, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, identifiers []string) ([]uint, error) {
		return nil, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filters fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return query, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) (map[string]uint, error) {
		return map[string]uint{"label1": uint(1)}, nil
	}

	testCases := []struct {
		name          string
		labels        []string
		expectedError string
	}{
		{
			name:          "no labels",
			labels:        []string{},
			expectedError: "",
		},
		{
			name:          "invalid label",
			labels:        []string{"iamnotalabel"},
			expectedError: "Invalid label name(s): iamnotalabel.",
		},
		{
			name:          "valid label",
			labels:        []string{"label1"},
			expectedError: "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: user})
			_, err := svc.NewDistributedQueryCampaignByIdentifiers(ctx, query.Query, nil, nil, tt.labels)

			if tt.expectedError == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}
