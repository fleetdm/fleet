package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestServiceSoftwareTitlesAuth(t *testing.T) {
	ds := new(mock.Store)

	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmf fleet.TeamFilter) ([]fleet.SoftwareTitle, int, *fleet.PaginationMetadata, error) {
		return []fleet.SoftwareTitle{}, 0, &fleet.PaginationMetadata{}, nil
	}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{}, nil
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
			premiumCtx := license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

			// List all software titles.
			_, _, _, err := svc.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// List software for a team.
			_, _, _, err = svc.ListSoftwareTitles(premiumCtx, fleet.SoftwareTitleListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			// List software for a team should fail no matter what
			// with a non-premium context
			if !tc.shouldFailTeamRead {
				_, _, _, err = svc.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
					TeamID: ptr.Uint(1),
				})
				require.ErrorContains(t, err, "Requires Fleet Premium license")
			}

			// Get a software title for a team
			_, err = svc.SoftwareTitleByID(ctx, 1, ptr.Uint(1))
			checkAuthErr(t, tc.shouldFailTeamRead, err)
		})
	}
}
