package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
		ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
			return nil, sql.ErrNoRows
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
			require.ErrorContains(t, err, "could not retrieve vpp token")
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
			require.ErrorContains(t, err, "could not retrieve vpp token")
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
		[]fleet.VPPAppTeam{{VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform}}},
		vppTokenInfo{Secret: "de-secret", Country: "de"},
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
	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, _ *uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{
			ID:          1,
			OrgName:     "us-org",
			Token:       "us-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "us",
		}, nil
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
	require.ErrorContains(t, err, "requested app not available on vpp account: "+adamID)
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
	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, _ *uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{
			ID:          1,
			OrgName:     "us-org",
			Token:       "us-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "us",
		}, nil
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
	require.ErrorContains(t, err, "requested app not available on vpp account: "+missing)
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

