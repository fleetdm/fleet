package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallersAuth(t *testing.T) {
	ds := new(mock.Store)

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license})

	testCases := []struct {
		name            string
		user            *fleet.User
		teamID          *uint
		shouldFailRead  bool
		shouldFailWrite bool
	}{
		{"no role no team", test.UserNoRoles, nil, true, true},
		{"no role team", test.UserNoRoles, ptr.Uint(1), true, true},
		{"global admin no team", test.UserAdmin, nil, false, false},
		{"global admin team", test.UserAdmin, ptr.Uint(1), false, false},
		{"global maintainer no team", test.UserMaintainer, nil, false, false},
		{"global maintainer team", test.UserMaintainer, ptr.Uint(1), false, false},
		{"global observer no team", test.UserObserver, nil, false, true},
		{"global observer team", test.UserObserver, ptr.Uint(1), false, true},
		{"global observer+ no team", test.UserObserverPlus, nil, false, true},
		{"global observer+ team", test.UserObserverPlus, ptr.Uint(1), false, true},
		{"global gitops no team", test.UserGitOps, nil, true, false},
		{"global gitops team", test.UserGitOps, ptr.Uint(1), true, false},
		{"team admin no team", test.UserTeamAdminTeam1, nil, true, true},
		{"team admin team", test.UserTeamAdminTeam1, ptr.Uint(1), false, false},
		{"team admin other team", test.UserTeamAdminTeam2, ptr.Uint(1), true, true},
		{"team maintainer no team", test.UserTeamMaintainerTeam1, nil, true, true},
		{"team maintainer team", test.UserTeamMaintainerTeam1, ptr.Uint(1), false, false},
		{"team maintainer other team", test.UserTeamMaintainerTeam2, ptr.Uint(1), true, true},
		{"team observer no team", test.UserTeamObserverTeam1, nil, true, true},
		{"team observer team", test.UserTeamObserverTeam1, ptr.Uint(1), false, true},
		{"team observer other team", test.UserTeamObserverTeam2, ptr.Uint(1), true, true},
		{"team observer+ no team", test.UserTeamObserverPlusTeam1, nil, true, true},
		{"team observer+ team", test.UserTeamObserverPlusTeam1, ptr.Uint(1), false, true},
		{"team observer+ other team", test.UserTeamObserverPlusTeam2, ptr.Uint(1), true, true},
		{"team gitops no team", test.UserTeamGitOpsTeam1, nil, true, true},
		{"team gitops team", test.UserTeamGitOpsTeam1, ptr.Uint(1), true, false},
		{"team gitops other team", test.UserTeamGitOpsTeam2, ptr.Uint(1), true, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScripts bool) (*fleet.SoftwareInstaller, error) {
				return &fleet.SoftwareInstaller{TeamID: tt.teamID}, nil
			}

			ds.DeleteSoftwareInstallerFunc = func(ctx context.Context, installerID uint) error {
				return nil
			}

			ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
				return nil
			}

			ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
				if tt.teamID != nil {
					return &fleet.Team{ID: *tt.teamID}, nil
				}

				return nil, nil
			}

			_, err := svc.DownloadSoftwareInstaller(ctx, 1, tt.teamID)
			if tt.teamID == nil {
				require.Error(t, err)
			} else {
				checkAuthErr(t, tt.shouldFailRead, err)
			}

			err = svc.DeleteSoftwareInstaller(ctx, 1, tt.teamID)
			if tt.teamID == nil {
				require.Error(t, err)
			} else {
				checkAuthErr(t, tt.shouldFailWrite, err)
			}

			// TODO: configure test with mock software installer store and add tests to check upload auth
		})
	}
}
