package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestBatchAssociateVPPApps(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	t.Run("Fails if missing VPP token when payloads to associate", func(t *testing.T) {
		ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
			return nil, nil
		}
		ds.GetAssignedVPPAppsFunc = func(ctx context.Context, teamID *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
			return nil, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		t.Run("dry run", func(t *testing.T) {
			_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
				{
					AppStoreID:       "my-fake-app",
					LabelsExcludeAny: []string{},
					LabelsIncludeAny: []string{},
					LabelsIncludeAll: []string{},
					Categories:       []string{},
					Platform:         fleet.MacOSPlatform,
				},
			}, true)
			require.ErrorContains(t, err, "No available VPP Token")
		})
		t.Run("not dry run", func(t *testing.T) {
			_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
				{
					AppStoreID:       "my-fake-app",
					LabelsExcludeAny: []string{},
					LabelsIncludeAny: []string{},
					LabelsIncludeAll: []string{},
					Categories:       []string{},
					Platform:         fleet.MacOSPlatform,
				},
			}, false)
			require.ErrorContains(t, err, "No available VPP Token")
		})
	})

	t.Run("Rejects malformed custom host vital reference in Android app configuration", func(t *testing.T) {
		ds.GetSoftwareCategoryNameToIDMapFunc = func(ctx context.Context, teamID uint, names []string) (map[string]uint, error) {
			return nil, nil
		}
		_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
			{
				AppStoreID:       "com.example.app",
				LabelsExcludeAny: []string{},
				LabelsIncludeAny: []string{},
				LabelsIncludeAll: []string{},
				Categories:       []string{},
				Platform:         fleet.AndroidPlatform,
				Configuration:    json.RawMessage(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_asset_tag"}}`),
			},
		}, true)
		var badReqErr *fleet.BadRequestError
		require.ErrorAs(t, err, &badReqErr)
		require.ErrorContains(t, err, "Invalid custom host vital reference")
	})

	t.Run("Rejects Android app configuration referencing an unknown custom host vital", func(t *testing.T) {
		ds.GetSoftwareCategoryNameToIDMapFunc = func(ctx context.Context, teamID uint, names []string) (map[string]uint, error) {
			return nil, nil
		}
		ds.ValidateReferencedCustomHostVitalsFunc = func(ctx context.Context, documents []string) error {
			return &fleet.MissingCustomHostVitalsError{MissingIDs: []uint{9}}
		}
		_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
			{
				AppStoreID:       "com.example.app",
				LabelsExcludeAny: []string{},
				LabelsIncludeAny: []string{},
				LabelsIncludeAll: []string{},
				Categories:       []string{},
				Platform:         fleet.AndroidPlatform,
				Configuration:    json.RawMessage(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_9"}}`),
			},
		}, true)
		var invalidArgErr *fleet.InvalidArgumentError
		require.ErrorAs(t, err, &invalidArgErr)
		require.ErrorContains(t, err, "is not defined")
	})

	t.Run("Android app configuration: infrastructure failure propagates instead of being reported as invalid input", func(t *testing.T) {
		ds.GetSoftwareCategoryNameToIDMapFunc = func(ctx context.Context, teamID uint, names []string) (map[string]uint, error) {
			return nil, nil
		}
		ds.ValidateReferencedCustomHostVitalsFunc = func(ctx context.Context, documents []string) error {
			return ctxerr.Wrap(ctx, errors.New("connection refused"), "validating custom host vitals")
		}
		_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
			{
				AppStoreID:       "com.example.app",
				LabelsExcludeAny: []string{},
				LabelsIncludeAny: []string{},
				LabelsIncludeAll: []string{},
				Categories:       []string{},
				Platform:         fleet.AndroidPlatform,
				Configuration:    json.RawMessage(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_9"}}`),
			},
		}, true)
		require.Error(t, err)
		require.ErrorContains(t, err, "connection refused")
		var invalidArgErr2 *fleet.InvalidArgumentError
		require.NotErrorAs(t, err, &invalidArgErr2, "an infrastructure failure must not be reported as invalid input (422)")
	})

	t.Run("Fails for Fleet Agent Android apps via GitOps", func(t *testing.T) {
		ds.GetSoftwareCategoryNameToIDMapFunc = func(ctx context.Context, teamID uint, names []string) (map[string]uint, error) {
			return nil, nil
		}

		fleetAgentPackages := []string{
			"com.fleetdm.agent",
			"com.fleetdm.agent.pingali",
			"com.fleetdm.agent.private.testuser",
		}

		for _, pkg := range fleetAgentPackages {
			t.Run(pkg+" dry run", func(t *testing.T) {
				_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
					{
						AppStoreID:       pkg,
						LabelsExcludeAny: []string{},
						LabelsIncludeAny: []string{},
						LabelsIncludeAll: []string{},
						Categories:       []string{},
						Platform:         fleet.AndroidPlatform,
					},
				}, true)
				require.ErrorContains(t, err, "The Fleet agent cannot be added manually")
			})
			t.Run(pkg+" not dry run", func(t *testing.T) {
				_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
					{
						AppStoreID:       pkg,
						LabelsExcludeAny: []string{},
						LabelsIncludeAny: []string{},
						LabelsIncludeAll: []string{},
						Categories:       []string{},
						Platform:         fleet.AndroidPlatform,
					},
				}, false)
				require.ErrorContains(t, err, "The Fleet agent cannot be added manually")
			})
		}
	})
}

// TestGetAnchoredVPPAppsMetadataSkipsReAnchorOnEmptyMetadata guards against
// the row mismatch where reAnchors holds an entry for a (adamID, platform)
// whose metadata fetch was skipped because Apple returned blanks. Before the
// fix the trailing UpdateVPPAppCountryCode in BatchAssociateVPPApps would
// rewrite the row's country without a matching metadata insert, leaving the
// row internally inconsistent until the next refresh.
func TestGetAnchoredVPPAppsMetadataSkipsReAnchorOnEmptyMetadata(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	// Fake Apple metadata endpoint that returns the requested adamID with a
	// blank Name, the documented transiently-degraded path that the second
	// loop's empty-metadata guard skips.
	metaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type plat struct {
			BundleID         string            `json:"bundleId"`
			Artwork          map[string]any    `json:"artwork"`
			LatestVersionRaw map[string]string `json:"latestVersionInfo"`
		}
		type attrs struct {
			Name           string          `json:"name"`
			DeviceFamilies []string        `json:"deviceFamilies"`
			Platforms      map[string]plat `json:"platformAttributes"`
		}
		type meta struct {
			ID         string `json:"id"`
			Attributes attrs  `json:"attributes"`
		}
		type resp struct {
			Data []meta `json:"data"`
		}
		out := resp{Data: []meta{{
			ID: "100",
			Attributes: attrs{
				Name:           "",
				DeviceFamilies: []string{"mac"},
				Platforms: map[string]plat{
					"osx": {
						BundleID:         "com.example.100",
						Artwork:          map[string]any{"url": "https://example.test/icon.png"},
						LatestVersionRaw: map[string]string{"versionDisplay": "1.0"},
					},
				},
			},
		}}}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(metaSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", metaSrv.URL, t)

	ds := new(mock.Store)
	// Existing row anchored to "us". The DE team adding it has no owning
	// token in the anchored country, so resolveAddAnchor returns
	// reAnchor=true with anchorCountry="de".
	ds.GetVPPAppByAdamIDPlatformFunc = func(ctx context.Context, adamID string, platform fleet.InstallableDevicePlatform) (*fleet.VPPApp, error) {
		return &fleet.VPPApp{
			VPPAppTeam:    fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: adamID, Platform: platform}},
			CountryCode:   "us",
			Name:          "Todoist US",
			LatestVersion: "0.1",
		}, nil
	}
	ds.GetVPPTokenOwningAppInCountryFunc = func(ctx context.Context, adamID string, platform fleet.InstallableDevicePlatform, country string) (*fleet.VPPTokenDB, error) {
		return nil, &batchNotFoundError{}
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{
		authz:  authorizer,
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		// Non-empty AppleConnectJWT so getVPPConfig's authenticator
		// short-circuits to the JWT instead of querying the datastore.
		config: config.FleetConfig{MDM: config.MDMConfig{AppleConnectJWT: "test-jwt"}},
	}

	apps, reAnchors, err := svc.getAnchoredVPPAppsMetadata(t.Context(),
		[]fleet.VPPAppTeam{{AdamID: "100", Platform: fleet.MacOSPlatform, VPPTokenID: new(uint(7))}},
		map[uint]vppTokenInfo{7: {Secret: "de-secret", Country: "de"}},
	)
	require.NoError(t, err)
	require.Empty(t, apps, "row with empty Apple metadata must not be inserted")
	require.Empty(t, reAnchors, "reAnchors must not contain entries for skipped rows")
}

// batchNotFoundError satisfies fleet.IsNotFound for the GetVPPTokenOwningAppInCountry mock.
type batchNotFoundError struct{}

func (batchNotFoundError) Error() string    { return "not found" }
func (batchNotFoundError) IsNotFound() bool { return true }

// TestGetAppStoreAppsDoesNotWriteMetadata guards the picker against writing
// the team's current-storefront metadata onto rows whose stored country
// is anchored elsewhere.
func TestGetAppStoreAppsDoesNotWriteMetadata(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, incompatible with t.Parallel.

	metaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type plat struct {
			BundleID         string            `json:"bundleId"`
			Artwork          map[string]any    `json:"artwork"`
			LatestVersionRaw map[string]string `json:"latestVersionInfo"`
		}
		type attrs struct {
			Name           string          `json:"name"`
			DeviceFamilies []string        `json:"deviceFamilies"`
			Platforms      map[string]plat `json:"platformAttributes"`
		}
		type meta struct {
			ID         string `json:"id"`
			Attributes attrs  `json:"attributes"`
		}
		type resp struct {
			Data []meta `json:"data"`
		}
		out := resp{Data: []meta{{
			ID: "100",
			Attributes: attrs{
				Name:           "Todoist DE",
				DeviceFamilies: []string{"mac"},
				Platforms: map[string]plat{
					"osx": {
						BundleID:         "com.example.100",
						Artwork:          map[string]any{"url": "https://example.test/de-icon.png"},
						LatestVersionRaw: map[string]string{"versionDisplay": "9.9"},
					},
				},
			},
		}}}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(metaSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", metaSrv.URL, t)

	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[{"adamId":"100","pricingParam":"STDQ"}]}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	teamID := uint(1)
	ds := new(mock.Store)
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{{
			ID:          1,
			OrgName:     "de-org",
			Token:       "de-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "de",
			Teams:       []fleet.TeamTuple{{ID: teamID}},
		}}, nil
	}
	// Existing row anchored to "us" while the team's current token is "de".
	ds.GetAssignedVPPAppsFunc = func(ctx context.Context, _ *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
		return map[fleet.VPPAppID]fleet.VPPAppTeam{
			{AdamID: "100", Platform: fleet.MacOSPlatform}: {
				VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform},
			},
		}, nil
	}
	batchInsertCalled := false
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, _ []*fleet.VPPApp) error {
		batchInsertCalled = true
		return nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{
		Service: &svcmock.Service{VerifyMDMAppleConfiguredFunc: func(context.Context) error { return nil }},
		authz:   authorizer,
		ds:      ds,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		config:  config.FleetConfig{MDM: config.MDMConfig{AppleConnectJWT: "test-jwt"}},
	}
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	apps, err := svc.GetAppStoreApps(ctx, &teamID, nil)
	require.NoError(t, err)
	require.Empty(t, apps, "already-assigned apps must be filtered out of the picker list")
	require.False(t, batchInsertCalled, "picker must not write metadata back")
}

// A no-platform numeric Adam ID expands to multiple (AdamID, platform)
// rows; the missing-asset error must surface each AdamID only once.
func TestBatchAssociateVPPAppsDedupsMissingAssetsError(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[]}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{{
			ID:          1,
			OrgName:     "us-org",
			Location:    "Acme US",
			Token:       "us-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "us",
			Teams:       []fleet.TeamTuple{},
		}}, nil
	}
	ds.GetAssignedVPPAppsFunc = func(ctx context.Context, teamID *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
		return nil, nil
	}
	ds.GetSoftwareCategoryNameToIDMapFunc = func(ctx context.Context, _ uint, _ []string) (map[string]uint, error) {
		return nil, nil
	}
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, values []string) ([]fleet.DuplicateStringGroup, error) {
		return nil, nil
	}

	svc := newTestService(t, ds)

	// ValidateSoftwareLabels inside the loop requires a present authz context
	// for Authorize to mark it checked.
	ctx := authz_ctx.NewContext(t.Context(), &authz_ctx.AuthorizationContext{})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	const adamID = "1107542306"
	_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
		{
			AppStoreID:       adamID,
			LabelsExcludeAny: []string{},
			LabelsIncludeAny: []string{},
			LabelsIncludeAll: []string{},
			Categories:       []string{},
			// Empty Platform triggers the auto-expansion to multiple
			// (AdamID, platform) rows — the multiplier this test guards.
		},
	}, false)

	require.Error(t, err)
	require.ErrorContains(t, err, `Couldn't apply. App "`+adamID+`" isn't available in the "Acme US" VPP token.`)
	require.Equal(t, 1, strings.Count(err.Error(), adamID),
		"missing-asset error must dedup by AdamID, got: %s", err.Error())
}

// A dry run must report an app that is missing from the VPP location the
// same way a real apply does. Before this test the dry run returned early,
// so an unlicensed or mistyped Adam ID passed `fleetctl gitops --dry-run`
// and only failed during the real apply.
func TestBatchAssociateVPPAppsDryRunReportsMissingAssets(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[{"adamId":"497799835"}]}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{{
			ID:          1,
			OrgName:     "us-org",
			Location:    "Acme US",
			Token:       "us-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "us",
			Teams:       []fleet.TeamTuple{},
		}}, nil
	}
	ds.GetAssignedVPPAppsFunc = func(ctx context.Context, teamID *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
		return nil, nil
	}
	ds.GetSoftwareCategoryNameToIDMapFunc = func(ctx context.Context, _ uint, _ []string) (map[string]uint, error) {
		return nil, nil
	}
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, _ []string) ([]fleet.DuplicateStringGroup, error) {
		return nil, nil
	}

	svc := newTestService(t, ds)

	ctx := authz_ctx.NewContext(t.Context(), &authz_ctx.AuthorizationContext{})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	payload := func(adamID string) []fleet.VPPBatchPayload {
		return []fleet.VPPBatchPayload{{
			AppStoreID:       adamID,
			Platform:         fleet.IOSPlatform,
			LabelsExcludeAny: []string{},
			LabelsIncludeAny: []string{},
			LabelsIncludeAll: []string{},
			Categories:       []string{},
		}}
	}

	// Licensed app: the dry run still succeeds without writing anything.
	_, _, err := svc.BatchAssociateVPPApps(ctx, "", payload("497799835"), true)
	require.NoError(t, err)
	require.False(t, ds.BatchInsertVPPAppsFuncInvoked)
	require.False(t, ds.SetTeamVPPAppsFuncInvoked)

	// Unlicensed app: the dry run fails with the same error as the real apply.
	const missing = "1107542306"
	_, _, err = svc.BatchAssociateVPPApps(ctx, "", payload(missing), true)
	require.Error(t, err)
	require.ErrorContains(t, err, `Couldn't apply. App "`+missing+`" isn't available in the "Acme US" VPP token.`)
	require.False(t, ds.BatchInsertVPPAppsFuncInvoked)
	require.False(t, ds.SetTeamVPPAppsFuncInvoked)
}

// A dry run for a team that does not exist yet used to return before any
// check. It must still refuse an Apple app that no VPP token can supply, since
// a new team's first real apply is where such an app otherwise fails, part-way
// through fleetctl's VPP re-apply.
func TestBatchAssociateVPPAppsDryRunNewTeamReportsMissingAssets(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[{"adamId":"497799835"}]}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, common_mysql.NotFound("Team")
	}
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, _ []string) ([]fleet.DuplicateStringGroup, error) {
		return nil, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{{
			ID:          1,
			OrgName:     "us-org",
			Token:       "us-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "us",
		}}, nil
	}

	svc := newTestService(t, ds)
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	payload := func(adamID string, platform fleet.InstallableDevicePlatform) []fleet.VPPBatchPayload {
		return []fleet.VPPBatchPayload{{
			AppStoreID:       adamID,
			Platform:         platform,
			LabelsExcludeAny: []string{},
			LabelsIncludeAny: []string{},
			LabelsIncludeAll: []string{},
			Categories:       []string{},
		}}
	}

	// Licensed on a token: the dry run passes and nothing is written.
	_, _, err := svc.BatchAssociateVPPApps(ctx, "New team", payload("497799835", fleet.IOSPlatform), true)
	require.NoError(t, err)

	// Licensed nowhere: the same error a real apply returns.
	const missing = "1107542306"
	_, _, err = svc.BatchAssociateVPPApps(ctx, "New team", payload(missing, fleet.IPadOSPlatform), true)
	require.Error(t, err)
	require.ErrorContains(t, err, "requested app not available on vpp account: "+missing)

	// Play Store apps are not VPP assets and are not checked here.
	_, _, err = svc.BatchAssociateVPPApps(ctx, "New team", payload("com.example.android", fleet.AndroidPlatform), true)
	require.NoError(t, err)

	require.False(t, ds.BatchInsertVPPAppsFuncInvoked)
	require.False(t, ds.SetTeamVPPAppsFuncInvoked)

	// A real apply for a missing team is unchanged: still a not-found error.
	_, _, err = svc.BatchAssociateVPPApps(ctx, "New team", payload(missing, fleet.IPadOSPlatform), false)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
}

// A dry run for a team that does not exist yet reads every unexpired token. A
// token Apple rejects must not fail the dry run while another token answers,
// since the team may never be assigned to it. Only when no token answers does
// the dry run fail, and when every token is expired it fails without calling Apple.
func TestBatchAssociateVPPAppsDryRunNewTeamToleratesUnreadableToken(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	var calls atomic.Int32
	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
		if r.Header.Get("Authorization") == "Bearer good" {
			_, _ = w.Write([]byte(`{"assets":[{"adamId":"497799835"}]}`))
			return
		}
		// Apple answers a revoked token with 200 and an error body.
		_, _ = w.Write([]byte(`{"errorNumber":9622,"errorMessage":"Invalid authentication token"}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	token := func(secret string, renew time.Time) *fleet.VPPTokenDB {
		return &fleet.VPPTokenDB{OrgName: secret + "-org", Token: secret, RenewDate: renew, CountryCode: "us"}
	}
	valid := time.Now().Add(24 * time.Hour)
	expired := time.Now().Add(-24 * time.Hour)

	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, common_mysql.NotFound("Team")
	}
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, _ []string) ([]fleet.DuplicateStringGroup, error) {
		return nil, nil
	}
	var tokens []*fleet.VPPTokenDB
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return tokens, nil
	}
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	payload := []fleet.VPPBatchPayload{{AppStoreID: "497799835", Platform: fleet.IOSPlatform}}

	// One revoked token and one that holds the app: the dry run passes.
	tokens = []*fleet.VPPTokenDB{token("bad", valid), token("good", valid)}
	_, _, err := svc.BatchAssociateVPPApps(ctx, "New team", payload, true)
	require.NoError(t, err)
	require.EqualValues(t, 2, calls.Load())

	// Only revoked tokens: the dry run fails with the fetch error.
	tokens = []*fleet.VPPTokenDB{token("bad", valid), token("worse", valid)}
	_, _, err = svc.BatchAssociateVPPApps(ctx, "New team", payload, true)
	require.ErrorContains(t, err, "unable to retrieve assets")
	require.ErrorContains(t, err, "Invalid authentication token")

	// Only expired tokens: the dry run fails before calling Apple.
	calls.Store(0)
	tokens = []*fleet.VPPTokenDB{token("good", expired)}
	_, _, err = svc.BatchAssociateVPPApps(ctx, "New team", payload, true)
	require.ErrorContains(t, err, "VPP token expired")
	require.Zero(t, calls.Load())

	// An expired token next to a valid one is ignored.
	tokens = []*fleet.VPPTokenDB{token("bad", expired), token("good", valid)}
	_, _, err = svc.BatchAssociateVPPApps(ctx, "New team", payload, true)
	require.NoError(t, err)
	require.EqualValues(t, 1, calls.Load())

	require.False(t, ds.BatchInsertVPPAppsFuncInvoked)
	require.False(t, ds.SetTeamVPPAppsFuncInvoked)
}

// A dry run for a team that does not exist yet must reject the same payloads
// the existing-team path rejects, instead of skipping them because no VPP
// lookup applies. Each case fails before any token is read.
func TestBatchAssociateVPPAppsDryRunNewTeamValidatesPayloads(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, common_mysql.NotFound("Team")
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return nil, nil
	}
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	ios := func(adamID, versionName string) fleet.VPPBatchPayload {
		return fleet.VPPBatchPayload{AppStoreID: adamID, Platform: fleet.IOSPlatform, VersionName: versionName}
	}
	tests := []struct {
		name string
		// duplicates is what the collation lookup reports, indexes into the
		// expanded version list.
		duplicates []fleet.DuplicateStringGroup
		payloads   []fleet.VPPBatchPayload
		wantErr    string
	}{
		{
			name:     "unsupported platform",
			payloads: []fleet.VPPBatchPayload{{AppStoreID: "not-an-adam-id", Platform: fleet.InstallableDevicePlatform("windows")}},
			wantErr:  "platform must be one of",
		},
		{
			name:     "fleet agent on android",
			payloads: []fleet.VPPBatchPayload{{AppStoreID: fleetAgentPackagePrefix + ".foo", Platform: fleet.AndroidPlatform}},
			wantErr:  "The Fleet agent cannot be added manually",
		},
		{
			name:     "empty category",
			payloads: []fleet.VPPBatchPayload{{AppStoreID: "497799835", Platform: fleet.IOSPlatform, Categories: []string{" "}}},
			wantErr:  "name is required",
		},
		{
			name:     "version name too long",
			payloads: []fleet.VPPBatchPayload{ios("497799835", strings.Repeat("v", fleet.MaxAppStoreAppVersionNameLength+1))},
			wantErr:  "The version name can't be longer than",
		},
		{
			name:     "entry with and without versions",
			payloads: []fleet.VPPBatchPayload{ios("497799835", ""), ios("497799835", "Stable")},
			wantErr:  "The app has an entry without versions and an entry with versions",
		},
		{
			name: "auto-update window without times",
			payloads: []fleet.VPPBatchPayload{{
				AppStoreID: "497799835", Platform: fleet.IOSPlatform, AutoUpdateEnabled: new(true),
			}},
			wantErr: "Start and end time must both be set",
		},
		{
			// No platform expands to macOS, iOS, and iPadOS, so the iOS window check applies.
			name: "auto-update window on an app with no platform",
			payloads: []fleet.VPPBatchPayload{{
				AppStoreID: "497799835", AutoUpdateStartTime: new("09:00"), AutoUpdateEndTime: new("09:30"),
			}},
			wantErr: "The update window must be at least one hour long",
		},
		{
			name:       "version names equal under collation",
			duplicates: []fleet.DuplicateStringGroup{{Indices: []int{0, 1}}},
			payloads:   []fleet.VPPBatchPayload{ios("497799835", "Stable"), ios("497799835", "stable")},
			wantErr:    "More than one version is named",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds.ListVPPTokensFuncInvoked = false
			ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, _ []string) ([]fleet.DuplicateStringGroup, error) {
				return tt.duplicates, nil
			}
			_, _, err := svc.BatchAssociateVPPApps(ctx, "New team", tt.payloads, true)
			require.ErrorContains(t, err, tt.wantErr)
			require.False(t, ds.ListVPPTokensFuncInvoked)
		})
	}
}

// A dry run for a team that does not exist yet reads the assets of every VPP
// token, so it must require global software write access first. Anyone who can
// read teams could otherwise probe which App Store IDs the org has licensed.
func TestBatchAssociateVPPAppsDryRunNewTeamRequiresWriteAccess(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, common_mysql.NotFound("Team")
	}
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, _ []string) ([]fleet.DuplicateStringGroup, error) {
		return nil, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return nil, nil
	}
	svc := newTestService(t, ds)

	payloads := []fleet.VPPBatchPayload{{
		AppStoreID:       "497799835",
		Platform:         fleet.IOSPlatform,
		LabelsExcludeAny: []string{},
		LabelsIncludeAny: []string{},
		LabelsIncludeAll: []string{},
		Categories:       []string{},
	}}

	team1 := func(role string) *fleet.User {
		return &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: role}}}
	}
	tests := []struct {
		name      string
		user      *fleet.User
		forbidden bool
	}{
		{"global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin)}, false},
		{"global maintainer", &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}, false},
		{"global gitops", &fleet.User{GlobalRole: new(fleet.RoleGitOps)}, false},
		{"global technician", &fleet.User{GlobalRole: new(fleet.RoleTechnician)}, true},
		{"global observer", &fleet.User{GlobalRole: new(fleet.RoleObserver)}, true},
		{"team admin", team1(fleet.RoleAdmin), true},
		{"team gitops", team1(fleet.RoleGitOps), true},
		{"team observer", team1(fleet.RoleObserver), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds.ListVPPTokensFuncInvoked = false
			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: tt.user})
			_, _, err := svc.BatchAssociateVPPApps(ctx, "New team", payloads, true)
			if tt.forbidden {
				var forbidden *authz.Forbidden
				require.ErrorAs(t, err, &forbidden)
				require.False(t, ds.ListVPPTokensFuncInvoked, "VPP tokens must not be read before authorization")
				return
			}
			require.NoError(t, err)
			require.True(t, ds.ListVPPTokensFuncInvoked)
		})
	}
}

// TestGetVPPTokensScoping verifies that GetVPPTokens returns every token to
// global readers but scopes the list to a team-scoped user's readable teams
// (plus "All teams" tokens), without leaking tokens from teams the user can't
// read. See #46057.
func TestGetVPPTokensScoping(t *testing.T) {
	ds := new(mock.Store)
	// Tokens: team 1, team 2, "All teams" (non-nil empty Teams), and an
	// unassigned token (nil Teams).
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{
			{ID: 1, OrgName: "team1", Teams: []fleet.TeamTuple{{ID: 1, Name: "Workstations"}}},
			{ID: 2, OrgName: "team2", Teams: []fleet.TeamTuple{{ID: 2, Name: "Servers"}}},
			{ID: 3, OrgName: "allteams", Teams: []fleet.TeamTuple{}},
			{ID: 4, OrgName: "unassigned", Teams: nil},
		}, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{
		authz:  authorizer,
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	globalMaintainer := &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}
	// Technician can read installable entities but not write them, so it must be
	// able to read the token list to use the picker (#46057 names this role).
	globalTechnician := &fleet.User{GlobalRole: new(fleet.RoleTechnician)}
	teamMaintainer1 := &fleet.User{Teams: []fleet.UserTeam{
		{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer},
	}}
	teamTechnician1 := &fleet.User{Teams: []fleet.UserTeam{
		{Team: fleet.Team{ID: 1}, Role: fleet.RoleTechnician},
	}}
	// Observer on the first team, maintainer on the second: must still be
	// authorized (via team 2) and scoped to team 2, never team 1.
	observerThenMaintainer := &fleet.User{Teams: []fleet.UserTeam{
		{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver},
		{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer},
	}}
	teamObserver1 := &fleet.User{Teams: []fleet.UserTeam{
		{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver},
	}}

	tests := []struct {
		name    string
		user    *fleet.User
		wantErr bool
		wantIDs []uint
	}{
		{"global admin sees all", &fleet.User{GlobalRole: new(fleet.RoleAdmin)}, false, []uint{1, 2, 3, 4}},
		{"global maintainer sees all", globalMaintainer, false, []uint{1, 2, 3, 4}},
		{"global technician sees all", globalTechnician, false, []uint{1, 2, 3, 4}},
		{"team maintainer scoped to team + all-teams", teamMaintainer1, false, []uint{1, 3}},
		{"team technician scoped to team + all-teams", teamTechnician1, false, []uint{1, 3}},
		{"observer-then-maintainer scoped to second team", observerThenMaintainer, false, []uint{2, 3}},
		{"team observer forbidden", teamObserver1, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: tt.user})
			got, err := svc.GetVPPTokens(ctx)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
				return
			}
			require.NoError(t, err)
			gotIDs := make([]uint, 0, len(got))
			for _, tok := range got {
				gotIDs = append(gotIDs, tok.ID)
			}
			require.ElementsMatch(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestAppStoreAppsRequireAppleMDM(t *testing.T) {
	t.Parallel()
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	setup := func(t *testing.T) (*Service, *mock.Store, *svcmock.Service) {
		ds := new(mock.Store)
		svc, baseSvc := newTestServiceWithMock(t, ds)
		baseSvc.VerifyMDMAppleConfiguredFunc = func(context.Context) error { return fleet.ErrMDMNotConfigured }
		ds.GetVPPTokenByTeamIDFunc = func(context.Context, *uint) (*fleet.VPPTokenDB, error) {
			return &fleet.VPPTokenDB{ID: 1}, nil
		}
		ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) {
			return []*fleet.VPPTokenDB{{ID: 1, Teams: []fleet.TeamTuple{}}}, nil
		}
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
		ds.TeamByNameFunc = func(context.Context, string) (*fleet.Team, error) {
			return nil, common_mysql.NotFound("Team")
		}
		ds.GetDuplicateStringGroupsUnderCollationFunc = func(context.Context, []string) ([]fleet.DuplicateStringGroup, error) {
			return nil, nil
		}
		return svc, ds, baseSvc
	}

	t.Run("listing VPP apps", func(t *testing.T) {
		svc, ds, _ := setup(t)
		_, err := svc.GetAppStoreApps(ctx, nil, nil)
		require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
		require.False(t, ds.ListVPPTokensFuncInvoked, "the VPP token must not be used")
	})

	t.Run("adding an Apple app", func(t *testing.T) {
		svc, _, _ := setup(t)
		_, err := svc.AddAppStoreApp(ctx, nil, fleet.VPPAppTeam{AdamID: "497799835", Platform: fleet.MacOSPlatform})
		require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	})

	t.Run("adding an Android app doesn't need Apple MDM", func(t *testing.T) {
		svc, _, baseSvc := setup(t)
		// Label validation runs after the Apple MDM check and fails in this test's context, so reaching it proves
		// the add got past the check.
		_, err := svc.AddAppStoreApp(ctx, nil, fleet.VPPAppTeam{AdamID: "com.example.app", Platform: fleet.AndroidPlatform})
		require.ErrorContains(t, err, "validating software labels")
		require.False(t, baseSvc.VerifyMDMAppleConfiguredFuncInvoked)
	})

	t.Run("GitOps apply with an Apple app writes nothing", func(t *testing.T) {
		svc, ds, _ := setup(t)
		ds.ListSoftwareCategoriesFunc = func(context.Context, uint) ([]fleet.SoftwareCategory, error) { return nil, nil }
		_, _, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
			{AppStoreID: "497799835", Platform: fleet.IOSPlatform, Categories: []string{"New category"}},
		}, false)
		require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
		require.False(t, ds.ListSoftwareCategoriesFuncInvoked, "categories must not be written for a rejected batch")
	})

	t.Run("GitOps dry run for a new fleet with an Apple app", func(t *testing.T) {
		svc, _, _ := setup(t)
		_, _, err := svc.BatchAssociateVPPApps(ctx, "New fleet", []fleet.VPPBatchPayload{{AppStoreID: "497799835", Platform: fleet.IOSPlatform}}, true)
		require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	})

	t.Run("GitOps dry run for a new fleet with only Android apps", func(t *testing.T) {
		svc, _, baseSvc := setup(t)
		_, _, err := svc.BatchAssociateVPPApps(ctx, "New fleet", []fleet.VPPBatchPayload{{AppStoreID: "com.example.app", Platform: fleet.AndroidPlatform}}, true)
		require.NoError(t, err)
		require.False(t, baseSvc.VerifyMDMAppleConfiguredFuncInvoked)
	})
}

func vppErrStatus(err error) int {
	var withStatus interface{ Status() int }
	if errors.As(err, &withStatus) {
		return withStatus.Status()
	}
	if withCode, ok := errors.AsType[fleet.ErrWithStatusCode](err); ok {
		return withCode.StatusCode()
	}
	if _, ok := errors.AsType[*fleet.InvalidArgumentError](err); ok {
		return http.StatusUnprocessableEntity
	}
	return 0
}

func TestResolveAppStoreAppToken(t *testing.T) {
	t.Parallel()

	valid := time.Now().Add(24 * time.Hour)
	us := &fleet.VPPTokenDB{ID: 1, Location: "Acme US", Token: "us-secret", RenewDate: valid, CountryCode: "us", Teams: []fleet.TeamTuple{{ID: 1}}}
	jp := &fleet.VPPTokenDB{ID: 2, Location: "Acme Japan", Token: "jp-secret", RenewDate: valid, CountryCode: "jp", Teams: []fleet.TeamTuple{{ID: 1}}}
	jpExpired := &fleet.VPPTokenDB{ID: 2, Location: "Acme Japan", Token: "jp-secret", RenewDate: time.Now().Add(-time.Hour), CountryCode: "jp", Teams: []fleet.TeamTuple{{ID: 1}}}
	unassigned := &fleet.VPPTokenDB{ID: 3, Location: "Unassigned", Token: "x", RenewDate: valid, CountryCode: "us"}
	otherFleet := &fleet.VPPTokenDB{ID: 4, Location: "Other fleet", Token: "x", RenewDate: valid, CountryCode: "us", Teams: []fleet.TeamTuple{{ID: 2}}}
	allFleets := &fleet.VPPTokenDB{ID: 5, Location: "All fleets", Token: "all-secret", RenewDate: valid, CountryCode: "us", Teams: []fleet.TeamTuple{}}
	noFleet := &fleet.VPPTokenDB{ID: 6, Location: "No fleet", Token: "no-secret", RenewDate: valid, CountryCode: "us", Teams: []fleet.TeamTuple{{ID: 0}}}

	tokenID := func(id uint) *uint { return &id }
	fleetID := func(id uint) *uint { return &id }

	maintainerOf := func(ids ...uint) *fleet.User {
		u := &fleet.User{}
		for _, id := range ids {
			u.Teams = append(u.Teams, fleet.UserTeam{ID: id, Role: fleet.RoleMaintainer})
		}
		return u
	}

	tests := []struct {
		name       string
		user       *fleet.User // global admin when nil
		tokens     []*fleet.VPPTokenDB
		teamID     *uint
		vppTokenID *uint
		wantID     uint
		wantErr    string
		wantStatus int
	}{
		{name: "single token without id", tokens: []*fleet.VPPTokenDB{us, otherFleet}, teamID: fleetID(1), wantID: 1},
		{name: "single token with its id", tokens: []*fleet.VPPTokenDB{us, otherFleet}, teamID: fleetID(1), vppTokenID: tokenID(1), wantID: 1},
		{name: "requested token of two", tokens: []*fleet.VPPTokenDB{us, jp}, teamID: fleetID(1), vppTokenID: tokenID(2), wantID: 2},
		{name: "all fleets token by id", tokens: []*fleet.VPPTokenDB{allFleets}, teamID: fleetID(7), vppTokenID: tokenID(5), wantID: 5},
		{name: "nil fleet is no fleet", tokens: []*fleet.VPPTokenDB{noFleet, us}, wantID: 6},
		{
			name: "missing token", tokens: []*fleet.VPPTokenDB{us}, teamID: fleetID(1), vppTokenID: tokenID(99),
			wantErr: "Couldn't list apps. VPP token with ID 99 doesn't exist.", wantStatus: http.StatusNotFound,
		},
		{
			name: "unassigned token", tokens: []*fleet.VPPTokenDB{us, unassigned}, teamID: fleetID(1), vppTokenID: tokenID(3),
			wantErr: `Couldn't list apps. VPP token "Unassigned" isn't assigned to this fleet.`, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "token assigned to another fleet", tokens: []*fleet.VPPTokenDB{us, otherFleet}, teamID: fleetID(1), vppTokenID: tokenID(4),
			wantErr: `Couldn't list apps. VPP token "Other fleet" isn't assigned to this fleet.`, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "token of another fleet hidden from fleet maintainer", user: maintainerOf(1), tokens: []*fleet.VPPTokenDB{us, otherFleet}, teamID: fleetID(1), vppTokenID: tokenID(4),
			wantErr: "Couldn't list apps. VPP token with ID 4 doesn't exist.", wantStatus: http.StatusNotFound,
		},
		{
			name: "token of another fleet visible to maintainer of both", user: maintainerOf(1, 2), tokens: []*fleet.VPPTokenDB{us, otherFleet}, teamID: fleetID(1), vppTokenID: tokenID(4),
			wantErr: `Couldn't list apps. VPP token "Other fleet" isn't assigned to this fleet.`, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "unassigned token hidden from fleet maintainer", user: maintainerOf(1), tokens: []*fleet.VPPTokenDB{us, unassigned}, teamID: fleetID(1), vppTokenID: tokenID(3),
			wantErr: "Couldn't list apps. VPP token with ID 3 doesn't exist.", wantStatus: http.StatusNotFound,
		},
		{
			name: "expired requested token", tokens: []*fleet.VPPTokenDB{us, jpExpired}, teamID: fleetID(1), vppTokenID: tokenID(2),
			wantErr: `Couldn't list apps. VPP token "Acme Japan" is expired. Please renew it and try again.`, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "expired only token", tokens: []*fleet.VPPTokenDB{jpExpired}, teamID: fleetID(1),
			wantErr: `Couldn't list apps. VPP token "Acme Japan" is expired. Please renew it and try again.`, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "two tokens without id", tokens: []*fleet.VPPTokenDB{us, jp}, teamID: fleetID(1),
			wantErr:    "Couldn't list apps. This fleet has more than one VPP token. Please specify vpp_token_id.",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "all fleets plus fleet token", tokens: []*fleet.VPPTokenDB{us, allFleets}, teamID: fleetID(1),
			wantErr:    "Couldn't list apps. This fleet has more than one VPP token. Please specify vpp_token_id.",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "all fleets plus no fleet token", tokens: []*fleet.VPPTokenDB{noFleet, allFleets}, teamID: fleetID(0),
			wantErr:    "Couldn't list apps. This fleet has more than one VPP token. Please specify vpp_token_id.",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "no token for the fleet", tokens: []*fleet.VPPTokenDB{otherFleet, unassigned}, teamID: fleetID(1),
			wantErr: "No available VPP Token", wantStatus: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) { return tt.tokens, nil }
			svc := newTestService(t, ds)

			user := tt.user
			if user == nil {
				user = &fleet.User{GlobalRole: new(fleet.RoleAdmin)}
			}
			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: user})

			tok, err := svc.resolveAppStoreAppToken(ctx, tt.teamID, tt.vppTokenID, "Couldn't list apps.")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Equal(t, tt.wantStatus, vppErrStatus(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, tok.ID)
		})
	}

	t.Run("add copy", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) { return []*fleet.VPPTokenDB{us, jp}, nil }
		svc := newTestService(t, ds)
		ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

		_, err := svc.resolveAppStoreAppToken(ctx, fleetID(1), nil, "Couldn't add software.")
		require.ErrorContains(t, err, "Couldn't add software. This fleet has more than one VPP token. Please specify vpp_token_id.")
		_, err = svc.resolveAppStoreAppToken(ctx, fleetID(1), tokenID(99), "Couldn't add software.")
		require.ErrorContains(t, err, "Couldn't add software. VPP token with ID 99 doesn't exist.")
	})
}

func TestGetAppStoreAppsUsesRequestedToken(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, incompatible with t.Parallel.
	var seenAuth []string
	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[]}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	valid := time.Now().Add(24 * time.Hour)
	ds := new(mock.Store)
	ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{
			{ID: 1, Location: "Acme US", Token: "us-secret", RenewDate: valid, CountryCode: "us", Teams: []fleet.TeamTuple{{ID: 1}}},
			{ID: 2, Location: "Acme Japan", Token: "jp-secret", RenewDate: valid, CountryCode: "jp", Teams: []fleet.TeamTuple{{ID: 1}}},
		}, nil
	}
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	teamID := uint(1)
	_, err := svc.GetAppStoreApps(ctx, &teamID, nil)
	require.ErrorContains(t, err, "Couldn't list apps. This fleet has more than one VPP token. Please specify vpp_token_id.")
	require.Empty(t, seenAuth, "Apple must not be called without a token choice")

	_, err = svc.GetAppStoreApps(ctx, &teamID, new(uint(2)))
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer jp-secret"}, seenAuth)
}

type addAppStoreAppFixture struct {
	svc        *Service
	ds         *mock.Store
	ctx        context.Context
	assets     string
	seenAuth   []string
	inserted   *fleet.VPPApp
	activities []fleet.ActivityDetails
}

// newAddAppStoreAppFixture wires AddAppStoreApp for an iOS app (adam ID "100")
// on the No fleet team, with fake Apple VPP and metadata servers.
func newAddAppStoreAppFixture(t *testing.T, tokens ...*fleet.VPPTokenDB) *addAppStoreAppFixture {
	t.Helper()
	fx := &addAppStoreAppFixture{assets: `{"assets":[{"adamId":"100","pricingParam":"STDQ"}]}`}

	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fx.seenAuth = append(fx.seenAuth, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fx.assets))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	metaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"100","attributes":{"name":"Canva","deviceFamilies":["iphone"],` +
			`"platformAttributes":{"ios":{"bundleId":"com.example.canva","artwork":{"url":"https://example.test/icon.png"},` +
			`"latestVersionInfo":{"versionDisplay":"1.0"}}}}}]}`))
	}))
	t.Cleanup(metaSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", metaSrv.URL, t)

	ds := new(mock.Store)
	svc, baseSvc := newTestServiceWithMock(t, ds)
	svc.config = config.FleetConfig{MDM: config.MDMConfig{AppleConnectJWT: "test-jwt"}}
	baseSvc.VerifyMDMAppleConfiguredFunc = func(context.Context) error { return nil }
	baseSvc.NewActivityFunc = func(_ context.Context, _ *fleet.User, a fleet.ActivityDetails) error {
		fx.activities = append(fx.activities, a)
		return nil
	}

	ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	ds.GetVPPTokenFunc = func(_ context.Context, id uint) (*fleet.VPPTokenDB, error) {
		for _, tok := range tokens {
			if tok.ID == id {
				return tok, nil
			}
		}
		return nil, common_mysql.NotFound("VPPToken")
	}
	ds.GetAppStoreAppVersionCountFunc = func(context.Context, *uint, fleet.VPPAppID, string) (uint, bool, error) {
		return 0, false, nil
	}
	ds.GetVPPAppByAdamIDPlatformFunc = func(context.Context, string, fleet.InstallableDevicePlatform) (*fleet.VPPApp, error) {
		return nil, common_mysql.NotFound("VPPApp")
	}
	ds.CheckConflictingInHouseAppExistsFunc = func(context.Context, *uint, string, string) (bool, error) { return false, nil }
	ds.InsertVPPAppWithTeamFunc = func(_ context.Context, app *fleet.VPPApp, _ *uint, _ *uint) (*fleet.VPPApp, error) {
		fx.inserted = app
		added := *app
		added.TitleID = 5
		added.VersionName = fleet.DefaultAppStoreAppVersionName
		return &added, nil
	}

	fx.svc = svc
	fx.ds = ds
	ctx := authz_ctx.NewContext(t.Context(), &authz_ctx.AuthorizationContext{})
	fx.ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	return fx
}

func TestAddAppStoreAppToken(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, incompatible with t.Parallel.
	valid := time.Now().Add(24 * time.Hour)
	// Tokens on the No fleet team (nil teamID).
	us := func() *fleet.VPPTokenDB {
		return &fleet.VPPTokenDB{ID: 1, Location: "Acme US", Token: "us-secret", RenewDate: valid, CountryCode: "us", Teams: []fleet.TeamTuple{{ID: 0}}}
	}
	jp := func() *fleet.VPPTokenDB {
		return &fleet.VPPTokenDB{ID: 2, Location: "Acme Japan", Token: "jp-secret", RenewDate: valid, CountryCode: "jp", Teams: []fleet.TeamTuple{{ID: 0}}}
	}
	iosApp := func(tokenID *uint) fleet.VPPAppTeam {
		return fleet.VPPAppTeam{AdamID: "100", Platform: fleet.IOSPlatform, VPPTokenID: tokenID}
	}

	t.Run("binds the requested token and records it on the activity", func(t *testing.T) {
		fx := newAddAppStoreAppFixture(t, us(), jp())
		_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(new(uint(2))))
		require.NoError(t, err)
		require.Equal(t, []string{"Bearer jp-secret"}, fx.seenAuth)
		require.NotNil(t, fx.inserted.VPPTokenID)
		require.Equal(t, uint(2), *fx.inserted.VPPTokenID)
		require.Equal(t, "jp", fx.inserted.CountryCode)

		require.Len(t, fx.activities, 1)
		act, ok := fx.activities[0].(fleet.ActivityAddedAppStoreApp)
		require.True(t, ok)
		require.Equal(t, &fleet.ActivityVPPToken{ID: 2, Location: "Acme Japan", CountryCode: "jp"}, act.VPPToken)
	})

	t.Run("single token binds without a token id", func(t *testing.T) {
		fx := newAddAppStoreAppFixture(t, us())
		_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(nil))
		require.NoError(t, err)
		require.Equal(t, uint(1), *fx.inserted.VPPTokenID)
		require.Equal(t, uint(1), fx.activities[0].(fleet.ActivityAddedAppStoreApp).VPPToken.ID)
	})

	t.Run("two tokens without a token id", func(t *testing.T) {
		fx := newAddAppStoreAppFixture(t, us(), jp())
		_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(nil))
		require.ErrorContains(t, err, "Couldn't add software. This fleet has more than one VPP token. Please specify vpp_token_id.")
		require.Equal(t, http.StatusUnprocessableEntity, vppErrStatus(err))
		require.False(t, fx.ds.InsertVPPAppWithTeamFuncInvoked)
	})

	t.Run("app missing from the token", func(t *testing.T) {
		fx := newAddAppStoreAppFixture(t, us(), jp())
		fx.assets = `{"assets":[]}`
		_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(new(uint(2))))
		require.ErrorContains(t, err, `Couldn't add software. "100" isn't available in the "Acme Japan" VPP token. Please purchase a license in Apple Business and try again.`)
		require.Equal(t, http.StatusUnprocessableEntity, vppErrStatus(err))
		require.False(t, fx.ds.InsertVPPAppWithTeamFuncInvoked)
	})

	t.Run("Android rejects a token id", func(t *testing.T) {
		fx := newAddAppStoreAppFixture(t, us(), jp())
		_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, fleet.VPPAppTeam{
			AdamID:     "com.example.app",
			Platform:   fleet.AndroidPlatform,
			VPPTokenID: new(uint(1)),
		})
		require.ErrorContains(t, err, "Couldn't add software. vpp_token_id isn't supported for Android apps.")
		require.Equal(t, http.StatusUnprocessableEntity, vppErrStatus(err))
		require.False(t, fx.ds.InsertVPPAppWithTeamFuncInvoked)
	})

	t.Run("second version", func(t *testing.T) {
		withExisting := func(t *testing.T, existingTokenID *uint) *addAppStoreAppFixture {
			fx := newAddAppStoreAppFixture(t, us(), jp())
			fx.ds.GetAppStoreAppVersionCountFunc = func(context.Context, *uint, fleet.VPPAppID, string) (uint, bool, error) {
				return 1, false, nil
			}
			fx.ds.GetVPPAppMetadataByAdamIDPlatformTeamIDFunc = func(context.Context, string, fleet.InstallableDevicePlatform, *uint) (*fleet.VPPApp, error) {
				return &fleet.VPPApp{Name: "Canva", VPPTokenID: existingTokenID}, nil
			}
			return fx
		}

		t.Run("inherits the first version's token", func(t *testing.T) {
			fx := withExisting(t, new(uint(2)))
			_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(nil))
			require.NoError(t, err)
			require.Equal(t, []string{"Bearer jp-secret"}, fx.seenAuth)
			require.Equal(t, uint(2), *fx.inserted.VPPTokenID)
		})

		t.Run("same token is accepted", func(t *testing.T) {
			fx := withExisting(t, new(uint(2)))
			_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(new(uint(2))))
			require.NoError(t, err)
			require.Equal(t, uint(2), *fx.inserted.VPPTokenID)
		})

		t.Run("another token conflicts", func(t *testing.T) {
			fx := withExisting(t, new(uint(2)))
			_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(new(uint(1))))
			require.ErrorContains(t, err, `Couldn't add software. Canva (iOS) is already added to this fleet from the "Acme Japan" VPP token.`)
			require.Equal(t, http.StatusConflict, vppErrStatus(err))
			require.Empty(t, fx.seenAuth)
			require.False(t, fx.ds.InsertVPPAppWithTeamFuncInvoked)
		})

		t.Run("a first version without a token falls through to resolution", func(t *testing.T) {
			fx := withExisting(t, nil)
			_, err := fx.svc.AddAppStoreApp(fx.ctx, nil, iosApp(new(uint(1))))
			require.NoError(t, err)
			require.Equal(t, uint(1), *fx.inserted.VPPTokenID)
		})
	})
}

func TestUpdateVPPTokenTeamsValidation(t *testing.T) {
	t.Parallel()
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	setup := func(t *testing.T) (*Service, *mock.Store) {
		ds := new(mock.Store)
		ds.GetVPPTokenFunc = func(_ context.Context, id uint) (*fleet.VPPTokenDB, error) {
			if id != 1 {
				return nil, common_mysql.NotFound("VPPToken")
			}
			return &fleet.VPPTokenDB{ID: 1}, nil
		}
		ds.TeamsSummaryFunc = func(context.Context) ([]*fleet.TeamSummary, error) {
			return []*fleet.TeamSummary{{ID: 1}, {ID: 2}}, nil
		}
		ds.UpdateVPPTokenTeamsFunc = func(_ context.Context, id uint, _ []uint) (*fleet.VPPTokenDB, error) {
			return &fleet.VPPTokenDB{ID: id}, nil
		}
		return newTestService(t, ds), ds
	}

	t.Run("missing token with fleets", func(t *testing.T) {
		svc, ds := setup(t)
		_, err := svc.UpdateVPPTokenTeams(ctx, 99, []uint{1})
		require.ErrorContains(t, err, "Couldn't edit. VPP token with ID 99 doesn't exist.")
		require.Equal(t, http.StatusNotFound, vppErrStatus(err))
		require.False(t, ds.UpdateVPPTokenTeamsFuncInvoked)
	})

	t.Run("missing token with null fleets", func(t *testing.T) {
		svc, ds := setup(t)
		_, err := svc.UpdateVPPTokenTeams(ctx, 99, nil)
		require.ErrorContains(t, err, "Couldn't edit. VPP token with ID 99 doesn't exist.")
		require.Equal(t, http.StatusNotFound, vppErrStatus(err))
		require.False(t, ds.UpdateVPPTokenTeamsFuncInvoked)
	})

	t.Run("missing fleet", func(t *testing.T) {
		svc, ds := setup(t)
		_, err := svc.UpdateVPPTokenTeams(ctx, 1, []uint{1, 7})
		require.ErrorContains(t, err, "Couldn't edit. Fleet with ID 7 doesn't exist.")
		require.Equal(t, http.StatusNotFound, vppErrStatus(err))
		require.False(t, ds.UpdateVPPTokenTeamsFuncInvoked)
	})

	t.Run("no fleet is always valid", func(t *testing.T) {
		svc, ds := setup(t)
		tok, err := svc.UpdateVPPTokenTeams(ctx, 1, []uint{0})
		require.NoError(t, err)
		require.Equal(t, uint(1), tok.ID)
		require.True(t, ds.UpdateVPPTokenTeamsFuncInvoked)
		require.False(t, ds.TeamsSummaryFuncInvoked, "fleets are only loaded to check non-zero ids")
	})

	t.Run("existing fleets and all fleets", func(t *testing.T) {
		svc, _ := setup(t)
		_, err := svc.UpdateVPPTokenTeams(ctx, 1, []uint{1, 2})
		require.NoError(t, err)
		_, err = svc.UpdateVPPTokenTeams(ctx, 1, []uint{})
		require.NoError(t, err)
	})
}

func TestUpdateAppStoreAppActivityToken(t *testing.T) {
	t.Parallel()
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	run := func(t *testing.T, tokenID *uint, getToken func(context.Context, uint) (*fleet.VPPTokenDB, error)) (*mock.Store, *fleet.ActivityEditedAppStoreApp, error) {
		ds := new(mock.Store)
		svc := newTestService(t, ds)
		ds.GetAppStoreAppVersionsByTeamAndTitleIDFunc = func(context.Context, uint, uint) ([]*fleet.VPPAppStoreApp, error) {
			return []*fleet.VPPAppStoreApp{{
				AdamID:         "100",
				Platform:       fleet.IOSPlatform,
				Name:           "Canva",
				VPPAppsTeamsID: 1,
				VPPTokenID:     tokenID,
			}}, nil
		}
		ds.InsertVPPAppWithTeamFunc = func(_ context.Context, app *fleet.VPPApp, _ *uint, _ *uint) (*fleet.VPPApp, error) {
			return &fleet.VPPApp{AppTeamID: 1}, nil
		}
		ds.GetVPPTokenFunc = getToken

		_, act, err := svc.UpdateAppStoreApp(ctx, 5, nil, fleet.AppStoreAppUpdatePayload{SelfService: new(true)})
		return ds, act, err
	}
	setup := func(t *testing.T, tokenID *uint, getToken func(context.Context, uint) (*fleet.VPPTokenDB, error)) *fleet.ActivityEditedAppStoreApp {
		_, act, err := run(t, tokenID, getToken)
		require.NoError(t, err)
		return act
	}

	t.Run("carries the app's token", func(t *testing.T) {
		act := setup(t, new(uint(2)), func(_ context.Context, id uint) (*fleet.VPPTokenDB, error) {
			return &fleet.VPPTokenDB{ID: id, Location: "Acme Japan", CountryCode: "jp"}, nil
		})
		require.Equal(t, &fleet.ActivityVPPToken{ID: 2, Location: "Acme Japan", CountryCode: "jp"}, act.VPPToken)
	})

	t.Run("app without a token", func(t *testing.T) {
		act := setup(t, nil, func(context.Context, uint) (*fleet.VPPTokenDB, error) {
			t.Fatal("no token to look up")
			return nil, nil
		})
		require.Nil(t, act.VPPToken)
	})

	t.Run("deleted token", func(t *testing.T) {
		act := setup(t, new(uint(2)), func(context.Context, uint) (*fleet.VPPTokenDB, error) {
			return nil, common_mysql.NotFound("VPPToken")
		})
		require.Nil(t, act.VPPToken)
	})

	t.Run("token lookup failure doesn't write the edit", func(t *testing.T) {
		ds, _, err := run(t, new(uint(2)), func(context.Context, uint) (*fleet.VPPTokenDB, error) {
			return nil, errors.New("db down")
		})
		require.ErrorContains(t, err, "db down")
		require.False(t, ds.InsertVPPAppWithTeamFuncInvoked)
	})
}

const batchVPPTeamName = "📱 Mobile"

// batchVPPFixture wires BatchAssociateVPPApps for fleet 7 with fake Apple VPP
// and metadata servers. Assets are per token secret.
type batchVPPFixture struct {
	svc *Service
	ds  *mock.Store
	ctx context.Context

	// assets maps a token secret to the adam IDs it owns.
	assets map[string][]string
	// assetCalls holds the secret of every assets request, in order.
	assetCalls []string
	// assigned and existing are the fleet's current rows and their vpp_apps rows.
	assigned map[fleet.VPPAppID]fleet.VPPAppTeam
	existing map[fleet.VPPAppID]*fleet.VPPApp

	setApps  []fleet.VPPAppTeam
	titleIDs map[string]uint
	inserted []*fleet.VPPApp
}

func newBatchVPPFixture(t *testing.T, tokens ...*fleet.VPPTokenDB) *batchVPPFixture {
	t.Helper()
	fx := &batchVPPFixture{
		assets:   map[string][]string{"us-secret": {"1", "2"}, "jp-secret": {"3"}},
		assigned: map[fleet.VPPAppID]fleet.VPPAppTeam{},
		existing: map[fleet.VPPAppID]*fleet.VPPApp{},
	}

	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		fx.assetCalls = append(fx.assetCalls, secret)
		var assets []string
		for _, id := range fx.assets[secret] {
			assets = append(assets, `{"adamId":"`+id+`","pricingParam":"STDQ"}`)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[` + strings.Join(assets, ",") + `]}`))
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	metaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var data []string
		for _, id := range []string{"1", "2", "3"} {
			data = append(data, `{"id":"`+id+`","attributes":{"name":"App `+id+`","deviceFamilies":["iphone","ipad"],`+
				`"platformAttributes":{"ios":{"bundleId":"com.example.`+id+`","artwork":{"url":"https://example.test/icon.png"},`+
				`"latestVersionInfo":{"versionDisplay":"1.0"}}}}}`)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[` + strings.Join(data, ",") + `]}`))
	}))
	t.Cleanup(metaSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", metaSrv.URL, t)

	ds := new(mock.Store)
	svc := newTestService(t, ds)
	svc.config = config.FleetConfig{MDM: config.MDMConfig{AppleConnectJWT: "test-jwt"}}

	ds.TeamByNameFunc = func(context.Context, string) (*fleet.Team, error) {
		return &fleet.Team{ID: 7, Name: batchVPPTeamName}, nil
	}
	ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	ds.GetAssignedVPPAppsFunc = func(context.Context, *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
		return fx.assigned, nil
	}
	ds.GetSoftwareCategoryNameToIDMapFunc = func(context.Context, uint, []string) (map[string]uint, error) { return nil, nil }
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(context.Context, []string) ([]fleet.DuplicateStringGroup, error) {
		return nil, nil
	}
	ds.GetVPPAppByAdamIDPlatformFunc = func(_ context.Context, adamID string, platform fleet.InstallableDevicePlatform) (*fleet.VPPApp, error) {
		if app, ok := fx.existing[fleet.VPPAppID{AdamID: adamID, Platform: platform}]; ok {
			return app, nil
		}
		return nil, common_mysql.NotFound("VPPApp")
	}
	ds.GetVPPTokenOwningAppInCountryFunc = func(_ context.Context, _ string, _ fleet.InstallableDevicePlatform, country string) (*fleet.VPPTokenDB, error) {
		for _, tok := range tokens {
			if tok.CountryCode == country {
				return tok, nil
			}
		}
		return nil, common_mysql.NotFound("VPPToken")
	}
	ds.BatchInsertVPPAppsFunc = func(_ context.Context, apps []*fleet.VPPApp) error {
		for i, app := range apps {
			app.TitleID = uint(100 + i)
		}
		fx.inserted = apps
		return nil
	}
	ds.SetTeamVPPAppsFunc = func(_ context.Context, _ *uint, apps []fleet.VPPAppTeam, titleIDs map[string]uint) (bool, error) {
		fx.setApps = apps
		fx.titleIDs = titleIDs
		return false, nil
	}
	ds.DeleteIconsAssociatedWithTitlesWithoutInstallersFunc = func(context.Context, uint) error { return nil }
	ds.GetVPPAppsFunc = func(context.Context, *uint) ([]fleet.VPPAppResponse, error) { return nil, nil }
	ds.GetEnterpriseFunc = func(context.Context) (*android.Enterprise, error) { return nil, common_mysql.NotFound("Enterprise") }

	fx.svc = svc
	fx.ds = ds
	ctx := authz_ctx.NewContext(t.Context(), &authz_ctx.AuthorizationContext{})
	fx.ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	return fx
}

func (fx *batchVPPFixture) apply(dryRun bool, payloads ...fleet.VPPBatchPayload) error {
	_, _, err := fx.svc.BatchAssociateVPPApps(fx.ctx, batchVPPTeamName, payloads, dryRun)
	return err
}

// boundTokens returns the token each app was passed to SetTeamVPPApps with.
func (fx *batchVPPFixture) boundTokens() map[fleet.VPPAppID]uint {
	bound := make(map[fleet.VPPAppID]uint, len(fx.setApps))
	for _, app := range fx.setApps {
		bound[app.VPPAppID] = *app.VPPTokenID
	}
	return bound
}

func batchUSToken() *fleet.VPPTokenDB {
	return &fleet.VPPTokenDB{ID: 1, Location: "Acme US", Token: "us-secret", RenewDate: time.Now().Add(24 * time.Hour), CountryCode: "us", Teams: []fleet.TeamTuple{{ID: 7}}}
}

func batchJPToken() *fleet.VPPTokenDB {
	return &fleet.VPPTokenDB{ID: 2, Location: "Acme Japan", Token: "jp-secret", RenewDate: time.Now().Add(24 * time.Hour), CountryCode: "jp", Teams: []fleet.TeamTuple{{ID: 7}}}
}

func batchIOS(adamID, location string) fleet.VPPBatchPayload {
	return fleet.VPPBatchPayload{
		AppStoreID:       adamID,
		Platform:         fleet.IOSPlatform,
		Location:         location,
		LabelsExcludeAny: []string{},
		LabelsIncludeAny: []string{},
		LabelsIncludeAll: []string{},
		Categories:       []string{},
	}
}

func batchAppID(adamID string) fleet.VPPAppID {
	return fleet.VPPAppID{AdamID: adamID, Platform: fleet.IOSPlatform}
}

func TestBatchAssociateVPPAppsTokenResolution(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	t.Run("binds the first-added token, or the located one", func(t *testing.T) {
		fx := newBatchVPPFixture(t, batchUSToken(), batchJPToken())
		require.NoError(t, fx.apply(false, batchIOS("1", ""), batchIOS("3", "Acme Japan")))
		require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("1"): 1, batchAppID("3"): 2}, fx.boundTokens())
		require.ElementsMatch(t, []string{"us-secret", "jp-secret"}, fx.assetCalls, "one assets call per token")
		require.Len(t, fx.inserted, 2)
	})

	t.Run("an existing app keeps its token without a location", func(t *testing.T) {
		fx := newBatchVPPFixture(t, batchUSToken(), batchJPToken())
		fx.assigned[batchAppID("3")] = fleet.VPPAppTeam{VPPTokenID: new(uint(2))}
		fx.existing[batchAppID("3")] = &fleet.VPPApp{Name: "App 3", TitleID: 33, CountryCode: "jp"}
		require.NoError(t, fx.apply(false, batchIOS("3", "")))
		require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("3"): 2}, fx.boundTokens())
		require.Equal(t, []string{"jp-secret"}, fx.assetCalls)
	})

	t.Run("an existing app accepts its own location", func(t *testing.T) {
		fx := newBatchVPPFixture(t, batchUSToken(), batchJPToken())
		fx.assigned[batchAppID("3")] = fleet.VPPAppTeam{VPPTokenID: new(uint(2))}
		fx.existing[batchAppID("3")] = &fleet.VPPApp{Name: "App 3", TitleID: 33, CountryCode: "jp"}
		require.NoError(t, fx.apply(false, batchIOS("3", "Acme Japan")))
		require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("3"): 2}, fx.boundTokens())
	})

	t.Run("a new version lands on the token of the app", func(t *testing.T) {
		fx := newBatchVPPFixture(t, batchUSToken(), batchJPToken())
		fx.assigned[batchAppID("3")] = fleet.VPPAppTeam{VPPTokenID: new(uint(2))}
		fx.existing[batchAppID("3")] = &fleet.VPPApp{Name: "App 3", TitleID: 33, CountryCode: "jp"}
		v1, v2 := batchIOS("3", ""), batchIOS("3", "")
		v1.VersionName, v2.VersionName = "Stable", "Beta"
		require.NoError(t, fx.apply(false, v1, v2))
		require.Len(t, fx.setApps, 2)
		for _, app := range fx.setApps {
			require.Equal(t, uint(2), *app.VPPTokenID)
		}
	})

	t.Run("a third token assigned to the fleet leaves apps on their tokens", func(t *testing.T) {
		third := &fleet.VPPTokenDB{ID: 3, Location: "Acme EU", Token: "eu-secret", RenewDate: time.Now().Add(24 * time.Hour), CountryCode: "de", Teams: []fleet.TeamTuple{{ID: 7}}}
		fx := newBatchVPPFixture(t, batchUSToken(), batchJPToken(), third)
		fx.assigned[batchAppID("1")] = fleet.VPPAppTeam{VPPTokenID: new(uint(1))}
		fx.assigned[batchAppID("3")] = fleet.VPPAppTeam{VPPTokenID: new(uint(2))}
		fx.existing[batchAppID("1")] = &fleet.VPPApp{Name: "App 1", TitleID: 11, CountryCode: "us"}
		fx.existing[batchAppID("3")] = &fleet.VPPApp{Name: "App 3", TitleID: 33, CountryCode: "jp"}
		require.NoError(t, fx.apply(false, batchIOS("1", ""), batchIOS("3", "")))
		require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("1"): 1, batchAppID("3"): 2}, fx.boundTokens())
	})

	t.Run("first-added is the lowest id among the fleet's and All fleets tokens", func(t *testing.T) {
		token := func(id uint, teams []fleet.TeamTuple) *fleet.VPPTokenDB {
			secret := "token-" + strings.Repeat("x", int(id))
			return &fleet.VPPTokenDB{ID: id, Location: secret, Token: secret, RenewDate: time.Now().Add(24 * time.Hour), CountryCode: "us", Teams: teams}
		}
		for _, tt := range []struct {
			name   string
			tokens []*fleet.VPPTokenDB
			want   uint
		}{
			{
				name:   "All fleets token below the fleet token",
				tokens: []*fleet.VPPTokenDB{token(1, []fleet.TeamTuple{{ID: 9}}), token(2, []fleet.TeamTuple{}), token(3, []fleet.TeamTuple{{ID: 7}})},
				want:   2,
			},
			{
				name:   "moving the lowest token to All fleets keeps it first",
				tokens: []*fleet.VPPTokenDB{token(1, []fleet.TeamTuple{}), token(2, []fleet.TeamTuple{{ID: 9}}), token(3, []fleet.TeamTuple{{ID: 7}})},
				want:   1,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				fx := newBatchVPPFixture(t, tt.tokens...)
				fx.assets = map[string][]string{}
				for _, tok := range tt.tokens {
					fx.assets[tok.Token] = []string{"1"}
				}
				require.NoError(t, fx.apply(false, batchIOS("1", "")))
				require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("1"): tt.want}, fx.boundTokens())
			})
		}
	})
}

func TestBatchAssociateVPPAppsTokenErrors(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	expired := time.Now().Add(-24 * time.Hour)
	missingLine := func(adamID, location string) string {
		return `Couldn't apply. App "` + adamID + `" isn't available in the "` + location + `" VPP token. Please purchase a license in Apple Business and try again.`
	}
	androidPayload := fleet.VPPBatchPayload{AppStoreID: "com.example.app", Platform: fleet.AndroidPlatform, Location: "Acme Japan"}

	tests := []struct {
		name     string
		tokens   func() (us, jp *fleet.VPPTokenDB)
		assigned map[fleet.VPPAppID]fleet.VPPAppTeam
		payloads []fleet.VPPBatchPayload
		want     string
		status   int
		// dryRunPasses is set for the one error a dry run skips.
		dryRunPasses bool
	}{
		{
			name:     "location doesn't match a token",
			payloads: []fleet.VPPBatchPayload{batchIOS("1", "Nowhere")},
			want:     `Couldn't apply. VPP token "Nowhere" doesn't exist.`,
			status:   http.StatusUnprocessableEntity,
		},
		{
			name: "location token isn't assigned to the fleet",
			tokens: func() (*fleet.VPPTokenDB, *fleet.VPPTokenDB) {
				jp := batchJPToken()
				jp.Teams = []fleet.TeamTuple{{ID: 9}}
				return batchUSToken(), jp
			},
			payloads:     []fleet.VPPBatchPayload{batchIOS("3", "Acme Japan")},
			want:         `Couldn't apply. VPP token "Acme Japan" isn't assigned to fleet "📱 Mobile". Add it under "org_settings.mdm.volume_purchasing_program".`,
			status:       http.StatusUnprocessableEntity,
			dryRunPasses: true,
		},
		{
			name:     "no location and the app isn't in the first-added token",
			payloads: []fleet.VPPBatchPayload{batchIOS("3", "")},
			want:     missingLine("3", "Acme US"),
			status:   http.StatusUnprocessableEntity,
		},
		{
			name:     "location and the app isn't in that token",
			payloads: []fleet.VPPBatchPayload{batchIOS("1", "Acme Japan")},
			want:     missingLine("1", "Acme Japan"),
			status:   http.StatusUnprocessableEntity,
		},
		{
			name: "one line per missing app, sorted, once per app across platforms",
			payloads: []fleet.VPPBatchPayload{
				{AppStoreID: "9", Categories: []string{}},
				batchIOS("3", ""),
				batchIOS("10", ""),
			},
			want:   strings.Join([]string{missingLine("10", "Acme US"), missingLine("3", "Acme US"), missingLine("9", "Acme US")}, "\n"),
			status: http.StatusUnprocessableEntity,
		},
		{
			name: "first-added token is expired for a new app",
			tokens: func() (*fleet.VPPTokenDB, *fleet.VPPTokenDB) {
				us := batchUSToken()
				us.RenewDate = expired
				return us, batchJPToken()
			},
			payloads: []fleet.VPPBatchPayload{batchIOS("1", "")},
			want:     `Couldn't apply. "Acme US" VPP token is expired. Please renew it.`,
			status:   http.StatusUnprocessableEntity,
		},
		{
			name: "location token is expired for a new app",
			tokens: func() (*fleet.VPPTokenDB, *fleet.VPPTokenDB) {
				jp := batchJPToken()
				jp.RenewDate = expired
				return batchUSToken(), jp
			},
			payloads: []fleet.VPPBatchPayload{batchIOS("3", "Acme Japan")},
			want:     `Couldn't apply. "Acme Japan" VPP token is expired. Please renew it.`,
			status:   http.StatusUnprocessableEntity,
		},
		{
			name:     "location differs from the token of an app that's already added",
			assigned: map[fleet.VPPAppID]fleet.VPPAppTeam{batchAppID("1"): {VPPTokenID: new(uint(1))}},
			payloads: []fleet.VPPBatchPayload{batchIOS("1", "Acme Japan")},
			want:     `Couldn't apply. Canva is already added from the "Acme US" VPP token. To add it from "Acme Japan", remove it, run GitOps, and then add it back with the new location.`,
			status:   http.StatusConflict,
		},
		{
			name: "no token available to the fleet",
			tokens: func() (*fleet.VPPTokenDB, *fleet.VPPTokenDB) {
				us, jp := batchUSToken(), batchJPToken()
				us.Teams, jp.Teams = []fleet.TeamTuple{{ID: 9}}, nil
				return us, jp
			},
			payloads: []fleet.VPPBatchPayload{batchIOS("1", "")},
			want:     "No available VPP Token",
			status:   http.StatusUnprocessableEntity,
		},
		{
			name:     "same app and platform with different locations",
			payloads: []fleet.VPPBatchPayload{batchIOS("3", "Acme Japan"), batchIOS("3", "")},
			want:     `Couldn't apply. App "3" is specified more than once with different location values.`,
			status:   http.StatusUnprocessableEntity,
		},
		{
			name:     "location on an Android app",
			payloads: []fleet.VPPBatchPayload{androidPayload},
			want:     `Couldn't apply. location isn't supported for Android apps.`,
			status:   http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		for _, dryRun := range []bool{false, true} {
			name := tt.name + " real run"
			if dryRun {
				name = tt.name + " dry run"
			}
			t.Run(name, func(t *testing.T) {
				tokens := func() (*fleet.VPPTokenDB, *fleet.VPPTokenDB) { return batchUSToken(), batchJPToken() }
				if tt.tokens != nil {
					tokens = tt.tokens
				}
				us, jp := tokens()
				fx := newBatchVPPFixture(t, us, jp)
				maps.Copy(fx.assigned, tt.assigned)
				for id := range tt.assigned {
					fx.existing[id] = &fleet.VPPApp{Name: "Canva", TitleID: 11, CountryCode: "us"}
				}

				err := fx.apply(dryRun, tt.payloads...)
				if dryRun && tt.dryRunPasses {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.want)
					require.Equal(t, tt.status, vppErrStatus(err))
				}
				require.False(t, fx.ds.SetTeamVPPAppsFuncInvoked)
			})
		}
	}

	t.Run("each missing app is reported once", func(t *testing.T) {
		fx := newBatchVPPFixture(t, batchUSToken(), batchJPToken())
		err := fx.apply(false, fleet.VPPBatchPayload{AppStoreID: "9", Categories: []string{}})
		require.Error(t, err)
		require.Equal(t, 1, strings.Count(err.Error(), `App "9"`))
	})
}

func TestBatchAssociateVPPAppsExistingAppOnExpiredToken(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	setup := func(t *testing.T) *batchVPPFixture {
		jp := batchJPToken()
		jp.RenewDate = time.Now().Add(-24 * time.Hour)
		fx := newBatchVPPFixture(t, batchUSToken(), jp)
		fx.assigned[batchAppID("3")] = fleet.VPPAppTeam{VPPTokenID: new(uint(2))}
		fx.existing[batchAppID("3")] = &fleet.VPPApp{Name: "App 3", TitleID: 33, CountryCode: "jp"}
		return fx
	}

	t.Run("survives a run next to a new app", func(t *testing.T) {
		fx := setup(t)
		require.NoError(t, fx.apply(false, batchIOS("1", ""), batchIOS("3", "")))
		require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("1"): 1, batchAppID("3"): 2}, fx.boundTokens(),
			"SetTeamVPPApps removes any app that isn't passed to it")
		require.Equal(t, []string{"us-secret"}, fx.assetCalls, "the expired token isn't read")
		require.Len(t, fx.inserted, 1, "metadata of the expired app is kept")
		require.Equal(t, uint(33), fx.titleIDs[batchAppID("3").String()])
		require.Equal(t, uint(100), fx.titleIDs[batchAppID("1").String()])
	})

	t.Run("survives a run with no other app", func(t *testing.T) {
		fx := setup(t)
		require.NoError(t, fx.apply(false, batchIOS("3", "")))
		require.Equal(t, map[fleet.VPPAppID]uint{batchAppID("3"): 2}, fx.boundTokens())
		require.False(t, fx.ds.BatchInsertVPPAppsFuncInvoked)
		require.Empty(t, fx.assetCalls)
	})

	t.Run("dry run doesn't fail or read the expired token", func(t *testing.T) {
		fx := setup(t)
		require.NoError(t, fx.apply(true, batchIOS("1", ""), batchIOS("3", "")))
		require.Equal(t, []string{"us-secret"}, fx.assetCalls)
	})

	t.Run("an app moved to the expired token's location still fails", func(t *testing.T) {
		fx := setup(t)
		err := fx.apply(false, batchIOS("1", "Acme Japan"))
		require.ErrorContains(t, err, `Couldn't apply. "Acme Japan" VPP token is expired. Please renew it.`)
	})
}

func TestBatchAssociateVPPAppsExpandedPlatformsOnExpiredToken(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	setup := func(t *testing.T) *batchVPPFixture {
		us := batchUSToken()
		us.RenewDate = time.Now().Add(-24 * time.Hour)
		fx := newBatchVPPFixture(t, us, batchJPToken())
		// An iOS-only app has no macOS row.
		for _, platform := range []fleet.InstallableDevicePlatform{fleet.IOSPlatform, fleet.IPadOSPlatform} {
			id := fleet.VPPAppID{AdamID: "1", Platform: platform}
			fx.assigned[id] = fleet.VPPAppTeam{VPPTokenID: new(uint(1))}
			fx.existing[id] = &fleet.VPPApp{Name: "App 1", TitleID: 11, CountryCode: "us"}
		}
		return fx
	}
	noPlatform := func(adamID string) fleet.VPPBatchPayload {
		return fleet.VPPBatchPayload{AppStoreID: adamID, Categories: []string{}}
	}

	t.Run("an app with no platform keeps its rows", func(t *testing.T) {
		fx := setup(t)
		require.NoError(t, fx.apply(false, noPlatform("1")))
		require.Equal(t, map[fleet.VPPAppID]uint{
			{AdamID: "1", Platform: fleet.IOSPlatform}:    1,
			{AdamID: "1", Platform: fleet.IPadOSPlatform}: 1,
		}, fx.boundTokens())
	})

	t.Run("an app with no platform and a location keeps its rows", func(t *testing.T) {
		fx := setup(t)
		located := noPlatform("1")
		located.Location = "Acme US"
		require.NoError(t, fx.apply(false, located))
		require.Equal(t, map[fleet.VPPAppID]uint{
			{AdamID: "1", Platform: fleet.IOSPlatform}:    1,
			{AdamID: "1", Platform: fleet.IPadOSPlatform}: 1,
		}, fx.boundTokens())
	})

	t.Run("a new app on the expired token still fails", func(t *testing.T) {
		fx := setup(t)
		err := fx.apply(false, noPlatform("2"))
		require.ErrorContains(t, err, `Couldn't apply. "Acme US" VPP token is expired. Please renew it.`)
	})

	t.Run("an explicit platform still fails", func(t *testing.T) {
		fx := setup(t)
		macOS := noPlatform("1")
		macOS.Platform = fleet.MacOSPlatform
		err := fx.apply(false, macOS)
		require.ErrorContains(t, err, `Couldn't apply. "Acme US" VPP token is expired. Please renew it.`)
	})
}

func TestBatchAssociateVPPAppsExpandedPlatformsInheritToken(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	// App 3 is iOS-only and sits on the Japan token, which isn't the first-added one.
	setup := func(t *testing.T, jp *fleet.VPPTokenDB) *batchVPPFixture {
		fx := newBatchVPPFixture(t, batchUSToken(), jp)
		for i, platform := range []fleet.InstallableDevicePlatform{fleet.IOSPlatform, fleet.IPadOSPlatform} {
			id := fleet.VPPAppID{AdamID: "3", Platform: platform}
			fx.assigned[id] = fleet.VPPAppTeam{AppTeamID: uint(5 + i), VPPTokenID: new(uint(2))}
			fx.existing[id] = &fleet.VPPApp{Name: "App 3", TitleID: 13, CountryCode: "jp"}
		}
		return fx
	}
	noPlatform := func(adamID string) fleet.VPPBatchPayload {
		return fleet.VPPBatchPayload{AppStoreID: adamID, Categories: []string{}}
	}
	want := map[fleet.VPPAppID]uint{
		{AdamID: "3", Platform: fleet.IOSPlatform}:    2,
		{AdamID: "3", Platform: fleet.IPadOSPlatform}: 2,
	}

	t.Run("an app with no platform and no location stays on its token", func(t *testing.T) {
		fx := setup(t, batchJPToken())
		require.NoError(t, fx.apply(false, noPlatform("3")))
		require.Equal(t, want, fx.boundTokens())
		require.Equal(t, []string{"jp-secret"}, fx.assetCalls)
	})

	t.Run("the token is expired", func(t *testing.T) {
		jp := batchJPToken()
		jp.RenewDate = time.Now().Add(-24 * time.Hour)
		fx := setup(t, jp)
		require.NoError(t, fx.apply(false, noPlatform("3")))
		require.Equal(t, want, fx.boundTokens())
		require.Empty(t, fx.assetCalls)
	})

	t.Run("a new app uses the first-added token", func(t *testing.T) {
		fx := setup(t, batchJPToken())
		require.NoError(t, fx.apply(false, noPlatform("2")))
		require.Equal(t, uint(1), fx.boundTokens()[fleet.VPPAppID{AdamID: "2", Platform: fleet.IOSPlatform}])
	})

	t.Run("a new app fails when the first-added token is expired", func(t *testing.T) {
		us := batchUSToken()
		us.RenewDate = time.Now().Add(-24 * time.Hour)
		fx := newBatchVPPFixture(t, us, batchJPToken())
		err := fx.apply(false, noPlatform("2"))
		require.ErrorContains(t, err, `Couldn't apply. "Acme US" VPP token is expired. Please renew it.`)
	})
}

func TestBatchAssociateVPPAppsDryRunNewFleetLocation(t *testing.T) {
	// dev_mode.SetOverride uses t.Setenv, which is incompatible with t.Parallel.

	setup := func(t *testing.T, tokens ...*fleet.VPPTokenDB) *batchVPPFixture {
		fx := newBatchVPPFixture(t, tokens...)
		fx.ds.TeamByNameFunc = func(context.Context, string) (*fleet.Team, error) { return nil, common_mysql.NotFound("Team") }
		return fx
	}

	t.Run("a located app is checked against that token only", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		require.NoError(t, fx.apply(true, batchIOS("3", "Acme Japan")))
		require.Equal(t, []string{"jp-secret"}, fx.assetCalls)
	})

	t.Run("a located app missing from its token", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		// The US token has app 1, but the location names Japan.
		err := fx.apply(true, batchIOS("1", "Acme Japan"))
		require.ErrorContains(t, err, `Couldn't apply. App "1" isn't available in the "Acme Japan" VPP token. Please purchase a license in Apple Business and try again.`)
		require.Equal(t, http.StatusUnprocessableEntity, vppErrStatus(err))
		require.Equal(t, []string{"jp-secret"}, fx.assetCalls)
	})

	t.Run("a location that doesn't exist", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		err := fx.apply(true, batchIOS("1", "Nowhere"))
		require.ErrorContains(t, err, `Couldn't apply. VPP token "Nowhere" doesn't exist.`)
		require.Empty(t, fx.assetCalls)
	})

	t.Run("a location with no tokens uploaded", func(t *testing.T) {
		fx := setup(t)
		err := fx.apply(true, batchIOS("1", "Acme Japan"))
		require.ErrorContains(t, err, `Couldn't apply. VPP token "Acme Japan" doesn't exist.`)
	})

	t.Run("an expired location token", func(t *testing.T) {
		jp := batchJPToken()
		jp.RenewDate = time.Now().Add(-24 * time.Hour)
		fx := setup(t, batchUSToken(), jp)
		err := fx.apply(true, batchIOS("3", "Acme Japan"))
		require.ErrorContains(t, err, `Couldn't apply. "Acme Japan" VPP token is expired. Please renew it.`)
		require.Empty(t, fx.assetCalls)
	})

	t.Run("apps without a location keep the union check", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		// Each app is on some token, so the union passes. App 3 is only on the Japan token.
		require.NoError(t, fx.apply(true, batchIOS("2", ""), batchIOS("3", "")))

		err := fx.apply(true, batchIOS("9", ""))
		require.ErrorContains(t, err, "requested app not available on vpp account: 9")
	})

	t.Run("located and unlocated apps are both checked", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		err := fx.apply(true, batchIOS("3", "Acme Japan"), batchIOS("9", ""))
		require.ErrorContains(t, err, "requested app not available on vpp account: 9")
		require.ElementsMatch(t, []string{"jp-secret", "us-secret"}, fx.assetCalls, "one assets call per token")
	})

	t.Run("a location on an Android app", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		err := fx.apply(true, fleet.VPPBatchPayload{AppStoreID: "com.example.app", Platform: fleet.AndroidPlatform, Location: "Acme Japan"})
		require.ErrorContains(t, err, "Couldn't apply. location isn't supported for Android apps.")
		require.False(t, fx.ds.ListVPPTokensFuncInvoked)
	})

	t.Run("one app with different locations", func(t *testing.T) {
		fx := setup(t, batchUSToken(), batchJPToken())
		err := fx.apply(true, batchIOS("3", "Acme Japan"), batchIOS("3", ""))
		require.ErrorContains(t, err, `Couldn't apply. App "3" is specified more than once with different location values.`)
		require.False(t, fx.ds.ListVPPTokensFuncInvoked)
	})
}

func TestResolveBatchVPPTokenStepOrder(t *testing.T) {
	t.Parallel()
	us, jp := batchUSToken(), batchJPToken()
	expiredUS := batchUSToken()
	expiredUS.RenewDate = time.Now().Add(-24 * time.Hour)
	all := []*fleet.VPPTokenDB{us, jp}

	tests := []struct {
		name       string
		teamTokens []*fleet.VPPTokenDB
		allTokens  []*fleet.VPPTokenDB
		existing   *uint
		location   string
		dryRun     bool
		wantID     uint
		wantErr    string
	}{
		{name: "unknown location beats an existing app", teamTokens: all, allTokens: all, existing: new(uint(1)), location: "Nowhere", wantErr: "doesn't exist"},
		{name: "unassigned beats the already-added error", teamTokens: []*fleet.VPPTokenDB{us}, allTokens: all, existing: new(uint(1)), location: "Acme Japan", wantErr: "isn't assigned to fleet"},
		{name: "a dry run skips the unassigned check", teamTokens: []*fleet.VPPTokenDB{us}, allTokens: all, location: "Acme Japan", dryRun: true, wantID: 2},
		{name: "a dry run still reaches the already-added error", teamTokens: []*fleet.VPPTokenDB{us}, allTokens: all, existing: new(uint(1)), location: "Acme Japan", dryRun: true, wantErr: "already added"},
		{name: "already added beats expiry", teamTokens: []*fleet.VPPTokenDB{expiredUS, jp}, allTokens: []*fleet.VPPTokenDB{expiredUS, jp}, existing: new(uint(1)), location: "Acme Japan", wantErr: "already added"},
		{name: "an existing app on an expired token passes", teamTokens: []*fleet.VPPTokenDB{expiredUS}, allTokens: []*fleet.VPPTokenDB{expiredUS}, existing: new(uint(1)), wantID: 1},
		{name: "an existing app outside the fleet's tokens keeps its token", teamTokens: []*fleet.VPPTokenDB{us}, allTokens: all, existing: new(uint(2)), wantID: 2},
		{name: "a new app on an expired token fails", teamTokens: []*fleet.VPPTokenDB{expiredUS}, allTokens: []*fleet.VPPTokenDB{expiredUS}, wantErr: "is expired"},
		{name: "a new app takes the first token", teamTokens: all, allTokens: all, wantID: 1},
		{name: "the location is compared after NFC normalization", teamTokens: all, allTokens: all, location: "Acme Japan", wantID: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := resolveBatchVPPToken(tt.teamTokens, tt.allTokens, tt.existing, tt.location, batchVPPTeamName, tt.dryRun)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, tok.ID)
		})
	}
}
