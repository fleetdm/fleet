package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestListVulnerabilities(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
		return []fleet.VulnerabilityWithMetadata{
			{
				CVEMeta: fleet.CVEMeta{
					CVE:         "CVE-2019-1234",
					Description: "A vulnerability",
				},
				CreatedAt:  time.Now(),
				HostsCount: 10,
			},
		}, nil, nil
	}

	t.Run("no list options", func(t *testing.T) {
		_, _, err := svc.ListVulnerabilities(ctx, fleet.VulnListOptions{})
		require.NoError(t, err)
	})

	t.Run("can only sort by supported columns", func(t *testing.T) {
		// invalid order key
		opts := fleet.VulnListOptions{ListOptions: fleet.ListOptions{
			OrderKey: "invalid",
		}, ValidSortColumns: freeValidVulnSortColumns}

		_, _, err := svc.ListVulnerabilities(ctx, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid order key")

		// valid order key
		opts.OrderKey = "cve"
		_, _, err = svc.ListVulnerabilities(ctx, opts)
		require.NoError(t, err)
	})
}

func TestVulnerabilitesAuth(t *testing.T) {
	ds := new(mock.Store)

	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
		return []fleet.VulnerabilityWithMetadata{}, &fleet.PaginationMetadata{}, nil
	}

	ds.VulnerabilityFunc = func(cxt context.Context, cve string, teamID *uint, includeCVEScores bool) (*fleet.VulnerabilityWithMetadata, error) {
		return &fleet.VulnerabilityWithMetadata{}, nil
	}

	ds.CountVulnerabilitiesFunc = func(cxt context.Context, opt fleet.VulnListOptions) (uint, error) {
		return 0, nil
	}

	ds.TeamExistsFunc = func(cxt context.Context, teamID uint) (bool, error) {
		return true, nil
	}

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
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tc.user})
			_, _, err := svc.ListVulnerabilities(ctx, fleet.VulnListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			_, _, err = svc.ListVulnerabilities(ctx, fleet.VulnListOptions{
				TeamID: 1,
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			_, err = svc.CountVulnerabilities(ctx, fleet.VulnListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			_, err = svc.CountVulnerabilities(ctx, fleet.VulnListOptions{
				TeamID: 1,
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			_, err = svc.Vulnerability(ctx, "CVE-2019-1234", nil, false)
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			_, err = svc.Vulnerability(ctx, "CVE-2019-1234", ptr.Uint(1), false)
			checkAuthErr(t, tc.shouldFailTeamRead, err)
		})
	}
}
