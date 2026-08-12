package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamScheduleAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, int, *fleet.PaginationMetadata, error) {
		return nil, 0, 0, nil, nil
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
			checkQueryWriteAuthErr(t, tt.shouldFailWrite, tt.shouldFailRead, err)

			err = svc.DeleteTeamScheduledQueries(ctx, 1, 99)
			checkQueryWriteAuthErr(t, tt.shouldFailWrite, tt.shouldFailRead, err)
		})
	}
}

// requireIndistinguishableNotFound asserts that the error for a report the
// caller may not see is identical to the error for one that doesn't exist.
// Matching HTTP status is not enough: the rendered body carries the
// NotFoundError's resource name, so a synthetic not-found that names a
// different resource than the datastore's real one still answers the
// attacker's question.
func requireIndistinguishableNotFound(t *testing.T, existingErr, missingErr error) {
	t.Helper()
	require.Error(t, existingErr)
	require.Error(t, missingErr)
	assert.True(t, fleet.IsNotFound(existingErr), "got: %v", existingErr)
	assert.True(t, fleet.IsNotFound(missingErr), "got: %v", missingErr)

	notFoundResource := func(err error) string {
		var notFound *common_mysql.NotFoundError
		if !errors.As(err, &notFound) || notFound == nil {
			return ""
		}
		return notFound.ResourceType
	}
	existingResource := notFoundResource(existingErr)
	require.NotEmpty(t, existingResource, "expected a NotFoundError, got: %v", existingErr)
	assert.Equal(t, notFoundResource(missingErr), existingResource,
		"the masked not-found must name the same resource as the datastore's real one")
}

func TestScheduleWritesAreBoundToPathScope(t *testing.T) {
	const (
		targetTeamID   = 1
		otherTeamID    = 3
		targetQueryID  = 10
		otherQueryID   = 30
		globalQueryID  = 70
		missingQueryID = 99
	)

	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		query := &fleet.Query{ID: id, Name: "foobar", Query: "SELECT 1;"}
		switch id {
		case targetQueryID:
			query.TeamID = new(uint(targetTeamID))
		case otherQueryID:
			query.TeamID = new(uint(otherTeamID))
		case globalQueryID:
			query.TeamID = nil
		default:
			// Must mirror the real datastore's wording (see the Query method in
			// datastore/mysql/queries.go): the masking is only airtight if the
			// synthetic not-found is byte-identical to the genuine one.
			return nil, common_mysql.NotFound("Report").WithID(id)
		}
		return query, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		return nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	globalAdminCtx := viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	otherTeamMaintainerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		Teams: []fleet.UserTeam{{Team: fleet.Team{ID: otherTeamID}, Role: fleet.RoleMaintainer}},
	}})

	t.Run("report from another fleet is not reachable through this fleet's path", func(t *testing.T) {
		_, err := svc.ModifyTeamScheduledQueries(globalAdminCtx, targetTeamID, otherQueryID, fleet.ScheduledQueryPayload{})
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)

		err = svc.DeleteTeamScheduledQueries(globalAdminCtx, targetTeamID, otherQueryID)
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)
	})

	t.Run("write access to a report does not make it reachable through another fleet's path", func(t *testing.T) {
		// This caller is a maintainer of the report's own fleet, so it may
		// legitimately modify and delete it -- but only
		// through that fleet's own path.
		// Addressing it through a fleet the caller has no access to must fail, or the path scope means nothing.
		_, err := svc.ModifyTeamScheduledQueries(otherTeamMaintainerCtx, targetTeamID, otherQueryID, fleet.ScheduledQueryPayload{})
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)

		err = svc.DeleteTeamScheduledQueries(otherTeamMaintainerCtx, targetTeamID, otherQueryID)
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)
	})

	t.Run("fleet maintainer still reaches its own report through its own path", func(t *testing.T) {
		// The counterpart to the subtest above: the same caller and the same
		// report, addressed through the fleet the report actually belongs to.
		_, err := svc.TeamScheduleQuery(otherTeamMaintainerCtx, otherTeamID, &fleet.ScheduledQuery{QueryID: globalQueryID, Interval: 10})
		require.NoError(t, err)

		_, err = svc.ModifyTeamScheduledQueries(otherTeamMaintainerCtx, otherTeamID, otherQueryID, fleet.ScheduledQueryPayload{})
		require.NoError(t, err)

		err = svc.DeleteTeamScheduledQueries(otherTeamMaintainerCtx, otherTeamID, otherQueryID)
		require.NoError(t, err)
	})

	t.Run("existing out-of-scope report is indistinguishable from a missing one", func(t *testing.T) {
		_, existingErr := svc.ModifyTeamScheduledQueries(otherTeamMaintainerCtx, targetTeamID, targetQueryID, fleet.ScheduledQueryPayload{})
		_, missingErr := svc.ModifyTeamScheduledQueries(otherTeamMaintainerCtx, targetTeamID, missingQueryID, fleet.ScheduledQueryPayload{})
		requireIndistinguishableNotFound(t, existingErr, missingErr)

		existingErr = svc.DeleteTeamScheduledQueries(otherTeamMaintainerCtx, targetTeamID, targetQueryID)
		missingErr = svc.DeleteTeamScheduledQueries(otherTeamMaintainerCtx, targetTeamID, missingQueryID)
		requireIndistinguishableNotFound(t, existingErr, missingErr)
	})

	t.Run("fleet report is not reachable through the global schedule path", func(t *testing.T) {
		_, err := svc.ModifyGlobalScheduledQueries(globalAdminCtx, targetQueryID, fleet.ScheduledQueryPayload{})
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)

		err = svc.DeleteGlobalScheduledQueries(globalAdminCtx, targetQueryID)
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)
	})

	t.Run("global report is not reachable through a fleet schedule path", func(t *testing.T) {
		_, err := svc.ModifyTeamScheduledQueries(globalAdminCtx, targetTeamID, globalQueryID, fleet.ScheduledQueryPayload{})
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)

		err = svc.DeleteTeamScheduledQueries(globalAdminCtx, targetTeamID, globalQueryID)
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err), "got: %v", err)
	})

	t.Run("schedule create does not disclose which source reports exist", func(t *testing.T) {
		// Only a global report can be scheduled, so a fleet-scoped source and a nonexistent one must answer alike.
		_, fleetScopedErr := svc.TeamScheduleQuery(globalAdminCtx, targetTeamID, &fleet.ScheduledQuery{QueryID: otherQueryID, Interval: 10})
		_, missingErr := svc.TeamScheduleQuery(globalAdminCtx, targetTeamID, &fleet.ScheduledQuery{QueryID: missingQueryID, Interval: 10})
		require.Error(t, fleetScopedErr)
		require.Error(t, missingErr)
		assert.True(t, fleet.IsNotFound(fleetScopedErr), "got: %v", fleetScopedErr)
		assert.True(t, fleet.IsNotFound(missingErr), "got: %v", missingErr)

		_, fleetScopedErr = svc.GlobalScheduleQuery(globalAdminCtx, &fleet.ScheduledQuery{QueryID: otherQueryID, Interval: 10})
		_, missingErr = svc.GlobalScheduleQuery(globalAdminCtx, &fleet.ScheduledQuery{QueryID: missingQueryID, Interval: 10})
		require.Error(t, fleetScopedErr)
		require.Error(t, missingErr)
		assert.True(t, fleet.IsNotFound(fleetScopedErr), "got: %v", fleetScopedErr)
		assert.True(t, fleet.IsNotFound(missingErr), "got: %v", missingErr)
	})

	t.Run("schedule create authorizes before touching the source report", func(t *testing.T) {
		// A caller who can't create a report on the target fleet must be
		// refused on that basis alone, without the source report ID reaching
		// the datastore to produce a distinguishable answer.
		ds.QueryFuncInvoked = false
		_, err := svc.TeamScheduleQuery(otherTeamMaintainerCtx, targetTeamID, &fleet.ScheduledQuery{QueryID: globalQueryID, Interval: 10})
		require.Error(t, err)
		var forbidden *authz.Forbidden
		require.ErrorAs(t, err, &forbidden)
		assert.False(t, ds.QueryFuncInvoked, "source report must not be looked up for an unauthorized caller")
	})

	t.Run("matching scope still works", func(t *testing.T) {
		_, err := svc.TeamScheduleQuery(globalAdminCtx, targetTeamID, &fleet.ScheduledQuery{QueryID: globalQueryID, Interval: 10})
		require.NoError(t, err)

		_, err = svc.GlobalScheduleQuery(globalAdminCtx, &fleet.ScheduledQuery{QueryID: globalQueryID, Interval: 10})
		require.NoError(t, err)

		_, err = svc.ModifyTeamScheduledQueries(globalAdminCtx, targetTeamID, targetQueryID, fleet.ScheduledQueryPayload{})
		require.NoError(t, err)

		err = svc.DeleteTeamScheduledQueries(globalAdminCtx, targetTeamID, targetQueryID)
		require.NoError(t, err)

		_, err = svc.ModifyGlobalScheduledQueries(globalAdminCtx, globalQueryID, fleet.ScheduledQueryPayload{})
		require.NoError(t, err)

		err = svc.DeleteGlobalScheduledQueries(globalAdminCtx, globalQueryID)
		require.NoError(t, err)
	})
}
