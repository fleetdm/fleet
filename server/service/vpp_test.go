package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/modules/activities"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func TestVPPAuth(t *testing.T) {
	ds := new(mock.Store)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetAndroidFleetServerSecret: {Value: []byte("secret")},
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return assets, nil
	}

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	androidMockClient := &android_mock.Client{}
	androidMockClient.SetAuthenticationSecretFunc = func(secret string) error { return nil }
	androidMockClient.EnterprisesWebAppsCreateFunc = func(ctx context.Context, enterpriseName string, app *androidmanagement.WebApp) (*androidmanagement.WebApp, error) {
		return &androidmanagement.WebApp{Name: "webapp1"}, nil
	}
	wlog := logging.NewJSONLogger(os.Stdout)
	activityModule := activities.NewActivityModule(ds, wlog)
	androidSvc, err := android_service.NewServiceWithClient(wlog.SlogLogger(), ds, androidMockClient, "test-private-key", ds, activityModule, config.AndroidAgentConfig{})
	require.NoError(t, err)

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, androidModule: androidSvc})

	// use a custom implementation of checkAuthErr as the service call will fail
	// with a different error for in case of authorization success and the
	// package-wide checkAuthErr requires no error.
	checkAuthErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
		} else if err != nil {
			require.NotEqual(t, (&authz.Forbidden{}).Error(), err.Error())
		}
	}

	testCases := []struct {
		name                   string
		user                   *fleet.User
		teamID                 *uint
		shouldFailRead         bool
		shouldFailWrite        bool
		shouldFailCreateWebApp bool
	}{
		{"no role no team", test.UserNoRoles, nil, true, true, true},
		{"no role team", test.UserNoRoles, ptr.Uint(1), true, true, true},
		{"global admin no team", test.UserAdmin, nil, false, false, false},
		{"global admin team", test.UserAdmin, ptr.Uint(1), false, false, false},
		{"global maintainer no team", test.UserMaintainer, nil, false, false, false},
		{"global maintainer team", test.UserMaintainer, ptr.Uint(1), false, false, false},
		{"global observer no team", test.UserObserver, nil, true, true, true},
		{"global observer team", test.UserObserver, ptr.Uint(1), true, true, true},
		{"global observer+ no team", test.UserObserverPlus, nil, true, true, true},
		{"global observer+ team", test.UserObserverPlus, ptr.Uint(1), true, true, true},
		{"global gitops no team", test.UserGitOps, nil, true, false, false},
		{"global gitops team", test.UserGitOps, ptr.Uint(1), true, false, false},
		{"team admin no team", test.UserTeamAdminTeam1, nil, true, true, false},
		{"team admin team", test.UserTeamAdminTeam1, ptr.Uint(1), false, false, false},
		{"team admin other team", test.UserTeamAdminTeam2, ptr.Uint(1), true, true, false},
		{"team maintainer no team", test.UserTeamMaintainerTeam1, nil, true, true, false},
		{"team maintainer team", test.UserTeamMaintainerTeam1, ptr.Uint(1), false, false, false},
		{"team maintainer other team", test.UserTeamMaintainerTeam2, ptr.Uint(1), true, true, false},
		{"team observer no team", test.UserTeamObserverTeam1, nil, true, true, true},
		{"team observer team", test.UserTeamObserverTeam1, ptr.Uint(1), true, true, true},
		{"team observer other team", test.UserTeamObserverTeam2, ptr.Uint(1), true, true, true},
		{"team observer+ no team", test.UserTeamObserverPlusTeam1, nil, true, true, true},
		{"team observer+ team", test.UserTeamObserverPlusTeam1, ptr.Uint(1), true, true, true},
		{"team observer+ other team", test.UserTeamObserverPlusTeam2, ptr.Uint(1), true, true, true},
		{"team gitops no team", test.UserTeamGitOpsTeam1, nil, true, true, false},
		{"team gitops team", test.UserTeamGitOpsTeam1, ptr.Uint(1), true, false, false},
		{"team gitops other team", test.UserTeamGitOpsTeam2, ptr.Uint(1), true, true, false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) {
				return false, nil
			}
			ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
				_ sqlx.QueryerContext,
			) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
				return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
			}
			ds.TeamWithExtrasFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
				return &fleet.Team{ID: 1}, nil
			}
			ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
				return &fleet.VPPTokenDB{ID: 1, OrgName: "org", Teams: []fleet.TeamTuple{{ID: 1}}}, nil
			}
			ds.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
				return &android.Enterprise{}, nil
			}

			// Note: these calls always return an error because they're attempting to unmarshal a
			// non-existent VPP token.
			_, err := svc.GetAppStoreApps(ctx, tt.teamID)
			if tt.teamID == nil {
				require.Error(t, err)
			} else {
				checkAuthErr(t, tt.shouldFailRead, err)
			}

			_, err = svc.AddAppStoreApp(ctx, tt.teamID, fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "123", Platform: fleet.IOSPlatform}})
			if tt.teamID == nil {
				require.Error(t, err)
			} else {
				checkAuthErr(t, tt.shouldFailWrite, err)
			}

			_, err = svc.CreateAndroidWebApp(ctx, "test", "http://example.com", nil)
			checkAuthErr(t, tt.shouldFailCreateWebApp, err)
		})
	}
}
