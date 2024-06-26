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

func TestService_ListSoftware(t *testing.T) {
	ds := new(mock.Store)

	var calledWithTeamID *uint
	var calledWithOpt fleet.SoftwareListOptions
	ds.ListSoftwareFunc = func(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, *fleet.PaginationMetadata, error) {
		calledWithTeamID = opt.TeamID
		calledWithOpt = opt
		return []fleet.Software{}, &fleet.PaginationMetadata{}, nil
	}

	user := &fleet.User{
		ID:         3,
		Email:      "foo@bar.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	_, _, err := svc.ListSoftware(ctx, fleet.SoftwareListOptions{TeamID: ptr.Uint(42), ListOptions: fleet.ListOptions{PerPage: 77, Page: 4}})
	require.NoError(t, err)

	assert.True(t, ds.ListSoftwareFuncInvoked)
	assert.Equal(t, ptr.Uint(42), calledWithTeamID)
	// sort order defaults to hosts_count descending, automatically, if not explicitly provided
	assert.Equal(t, fleet.ListOptions{PerPage: 77, Page: 4, OrderKey: "hosts_count", OrderDirection: fleet.OrderDescending}, calledWithOpt.ListOptions)
	assert.True(t, calledWithOpt.WithHostCounts)

	// call again, this time with an explicit sort
	ds.ListSoftwareFuncInvoked = false
	_, _, err = svc.ListSoftware(ctx, fleet.SoftwareListOptions{TeamID: nil, ListOptions: fleet.ListOptions{PerPage: 11, Page: 2, OrderKey: "id", OrderDirection: fleet.OrderAscending}})
	require.NoError(t, err)

	assert.True(t, ds.ListSoftwareFuncInvoked)
	assert.Nil(t, calledWithTeamID)
	assert.Equal(t, fleet.ListOptions{PerPage: 11, Page: 2, OrderKey: "id", OrderDirection: fleet.OrderAscending}, calledWithOpt.ListOptions)
	assert.True(t, calledWithOpt.WithHostCounts)
}

func TestServiceSoftwareInventoryAuth(t *testing.T) {
	ds := new(mock.Store)

	ds.ListSoftwareFunc = func(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, *fleet.PaginationMetadata, error) {
		return []fleet.Software{}, &fleet.PaginationMetadata{}, nil
	}
	ds.CountSoftwareFunc = func(ctx context.Context, opt fleet.SoftwareListOptions) (int, error) {
		return 0, nil
	}
	ds.SoftwareByIDFunc = func(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *fleet.TeamFilter) (*fleet.Software, error) {
		return &fleet.Software{}, nil
	}
	ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) { return true, nil }
	svc, ctx := newTestService(t, ds, nil, nil)

	for _, tc := range []struct {
		name                 string
		user                 *fleet.User
		shouldFailGlobalRead bool
		shouldFailTeamRead   bool
	}{
		{
			name: "global-admin",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name: "global-maintainer",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleMaintainer),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name: "global-observer",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleObserver),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-admin-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleAdmin,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-maintainer-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleMaintainer,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-observer-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleObserver,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-admin-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleAdmin,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name: "team-maintainer-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleMaintainer,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name: "team-observer-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleObserver,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tc.user})

			// List all software.
			_, _, err := svc.ListSoftware(ctx, fleet.SoftwareListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// Count all software.
			_, err = svc.CountSoftware(ctx, fleet.SoftwareListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// List software for a team.
			_, _, err = svc.ListSoftware(ctx, fleet.SoftwareListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			// Count software for a team.
			_, err = svc.CountSoftware(ctx, fleet.SoftwareListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			_, err = svc.SoftwareByID(ctx, 1, ptr.Uint(1), false)
			checkAuthErr(t, tc.shouldFailTeamRead, err)
		})
	}
}
