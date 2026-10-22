package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
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
	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, _ *uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{
			ID:          1,
			OrgName:     "de-org",
			Token:       "de-secret",
			RenewDate:   time.Now().Add(24 * time.Hour),
			CountryCode: "de",
		}, nil
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

	apps, err := svc.GetAppStoreApps(ctx, &teamID)
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
		_, err := svc.GetAppStoreApps(ctx, nil)
		require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
		require.False(t, ds.GetVPPTokenByTeamIDFuncInvoked, "the VPP token must not be used")
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

// testAppStoreAppDatastore keeps, in memory, the state the App Store app configuration reconciler reads and the installs it
// writes. Tests change the state the way Fleet does, run the cron and check which version each host was sent.
type testAppStoreAppDatastore struct {
	ds  *mock.Store
	now time.Time

	hosts             []*fleet.Host
	mdmOffHostIDs     map[uint]struct{}
	inventoryHostIDs  map[uint]struct{}
	labelIDsByHostID  map[uint]map[uint]struct{}
	versionsByFleetID map[uint][]*fleet.VPPAppStoreApp
	tokenByFleetID    map[uint]*fleet.VPPTokenDB
	cursor            uint

	installs              []fleet.HostVPPSoftwareInstall
	failedInstalls        []fleet.HostVPPSoftwareInstall
	queuedVersionByHostID map[uint]uint
	queueInstalls         bool
	rejectedHostIDs       map[uint]struct{}
	sentVersionByHostID   map[uint]uint
}

func (f *testAppStoreAppDatastore) GetVPPTokenIfCanInstallVPPApps(ctx context.Context, appleDevice bool, host *fleet.Host) (string, error) {
	return "token", nil
}

func (f *testAppStoreAppDatastore) InstallVPPAppPostValidation(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, token string, opts fleet.HostSoftwareInstallOptions) (string, error) {
	if host == nil || vppApp == nil {
		return "", errors.New("missing host or app")
	}
	if _, rejected := f.rejectedHostIDs[host.ID]; rejected {
		return "", errors.New("Apple rejected the install")
	}
	f.sentVersionByHostID[host.ID] = vppApp.AppTeamID
	if f.queueInstalls {
		f.queuedVersionByHostID[host.ID] = vppApp.AppTeamID
		return "command-uuid", nil
	}
	f.installs = append(f.installs, fleet.HostVPPSoftwareInstall{HostID: host.ID, VPPAppTeamID: new(vppApp.AppTeamID), CreatedAt: f.now})
	return "command-uuid", nil
}

func TestReconcileAppStoreAppConfigurations(t *testing.T) {
	ctx := license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	nothingSent := map[uint]uint{}

	// tick moves the test datastore's clock forward a second, so each change and install gets its own whole second like in MySQL
	tick := func(f *testAppStoreAppDatastore) {
		f.now = f.now.Add(time.Second)
	}
	hostByID := func(f *testAppStoreAppDatastore, hostID uint) *fleet.Host {
		for _, host := range f.hosts {
			if host.ID == hostID {
				return host
			}
		}
		return &fleet.Host{}
	}
	versionByID := func(f *testAppStoreAppDatastore, versionID uint) *fleet.VPPAppStoreApp {
		for _, versions := range f.versionsByFleetID {
			for _, version := range versions {
				if version.VPPAppsTeamsID == versionID {
					return version
				}
			}
		}
		return nil
	}
	inScope := func(f *testAppStoreAppDatastore, hostID uint, version *fleet.VPPAppStoreApp) bool {
		if version == nil {
			return false
		}
		labelIDs := f.labelIDsByHostID[hostID]
		for _, label := range version.LabelsExcludeAny {
			if _, member := labelIDs[label.LabelID]; member {
				return false
			}
		}
		for _, label := range version.LabelsIncludeAll {
			if _, member := labelIDs[label.LabelID]; !member {
				return false
			}
		}
		if len(version.LabelsIncludeAny) == 0 {
			return true
		}
		for _, label := range version.LabelsIncludeAny {
			if _, member := labelIDs[label.LabelID]; member {
				return true
			}
		}
		return false
	}

	newTestAppStoreAppDatastore := func() *testAppStoreAppDatastore {
		f := &testAppStoreAppDatastore{
			ds:                    new(mock.Store),
			now:                   time.Now().Add(-24 * time.Hour).Truncate(time.Second),
			mdmOffHostIDs:         map[uint]struct{}{},
			inventoryHostIDs:      map[uint]struct{}{},
			labelIDsByHostID:      map[uint]map[uint]struct{}{},
			versionsByFleetID:     map[uint][]*fleet.VPPAppStoreApp{},
			tokenByFleetID:        map[uint]*fleet.VPPTokenDB{},
			queuedVersionByHostID: map[uint]uint{},
			rejectedHostIDs:       map[uint]struct{}{},
		}
		f.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
		}
		f.ds.GetAppStoreAppConfigurationReconcileCursorFunc = func(ctx context.Context) (uint, error) {
			return f.cursor, nil
		}
		f.ds.SetAppStoreAppConfigurationReconcileCursorFunc = func(ctx context.Context, cursor uint) error {
			f.cursor = cursor
			return nil
		}
		f.ds.ListIOSAndIPadOSMDMHostsAfterIDFunc = func(ctx context.Context, afterHostID uint, limit int) ([]*fleet.Host, error) {
			var page []*fleet.Host
			for _, host := range f.hosts {
				if _, mdmOff := f.mdmOffHostIDs[host.ID]; mdmOff || host.ID <= afterHostID || len(page) == limit {
					continue
				}
				page = append(page, host)
			}
			return page, nil
		}
		f.ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
			token, ok := f.tokenByFleetID[ptr.ValOrZero(teamID)]
			if !ok {
				return nil, &notFoundError{}
			}
			return token, nil
		}
		f.ds.GetAppStoreAppVersionsByTeamFunc = func(ctx context.Context, fleetID uint) (map[uint][]*fleet.VPPAppStoreApp, error) {
			versionsByTitleID := make(map[uint][]*fleet.VPPAppStoreApp)
			if len(f.versionsByFleetID[fleetID]) > 0 {
				versionsByTitleID[1] = f.versionsByFleetID[fleetID]
			}
			return versionsByTitleID, nil
		}
		f.ds.ListHostAppStoreAppInstallVersionsFunc = func(ctx context.Context, fleetID uint, hostIDs []uint) (map[uint]map[uint]fleet.HostVPPSoftwareInstall, error) {
			installByHostID := make(map[uint]fleet.HostVPPSoftwareInstall)
			for _, hostID := range hostIDs {
				if _, inInventory := f.inventoryHostIDs[hostID]; !inInventory || ptr.ValOrZero(hostByID(f, hostID).TeamID) != fleetID {
					continue
				}
				for _, install := range f.installs {
					if install.HostID == hostID {
						installByHostID[hostID] = install
					}
				}
			}
			return map[uint]map[uint]fleet.HostVPPSoftwareInstall{1: installByHostID}, nil
		}
		f.ds.GetIncludedHostIDMapForVPPAppHostsFunc = func(ctx context.Context, vppAppTeamID uint, hostIDs []uint) (map[uint]struct{}, error) {
			version := versionByID(f, vppAppTeamID)
			included := make(map[uint]struct{})
			for _, hostID := range hostIDs {
				if inScope(f, hostID, version) {
					included[hostID] = struct{}{}
				}
			}
			return included, nil
		}
		f.ds.GetHostIDsWithUnactivatedVPPAppInstallFunc = func(ctx context.Context, vppAppTeamID uint, hostIDs []uint) (map[uint]struct{}, error) {
			queued := make(map[uint]struct{})
			for _, hostID := range hostIDs {
				if versionID, ok := f.queuedVersionByHostID[hostID]; ok && versionID == vppAppTeamID {
					queued[hostID] = struct{}{}
				}
			}
			return queued, nil
		}
		f.ds.GetHostIDsWithRecentFailedVPPAppInstallFunc = func(ctx context.Context, vppAppTeamID uint, since time.Time, hostIDs []uint) (map[uint]struct{}, error) {
			failed := make(map[uint]struct{})
			for _, install := range f.failedInstalls {
				if ptr.ValOrZero(install.VPPAppTeamID) == vppAppTeamID && !install.CreatedAt.Before(since) && slices.Contains(hostIDs, install.HostID) {
					failed[install.HostID] = struct{}{}
				}
			}
			return failed, nil
		}
		f.ds.GetVPPAppByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, vppAppTeamID uint) (*fleet.VPPApp, error) {
			version := versionByID(f, vppAppTeamID)
			if version == nil {
				return nil, &notFoundError{}
			}
			return &fleet.VPPApp{VPPAppID: version.VPPAppID, AppTeamID: vppAppTeamID}, nil
		}
		return f
	}

	addFleet := func(f *testAppStoreAppDatastore, fleetID uint) {
		f.tokenByFleetID[fleetID] = &fleet.VPPTokenDB{Token: "token", RenewDate: time.Now().Add(24 * time.Hour)}
	}
	addHost := func(f *testAppStoreAppDatastore, hostID uint, fleetID uint, labelIDs ...uint) *fleet.Host {
		host := &fleet.Host{ID: hostID, UUID: fmt.Sprintf("host-%d", hostID), Platform: "ios", TeamID: ptr.UintOrNilIfZero(fleetID)}
		f.hosts = append(f.hosts, host)
		f.inventoryHostIDs[hostID] = struct{}{}
		f.labelIDsByHostID[hostID] = map[uint]struct{}{}
		for _, labelID := range labelIDs {
			f.labelIDsByHostID[hostID][labelID] = struct{}{}
		}
		return host
	}
	addVersion := func(f *testAppStoreAppDatastore, fleetID uint, versionID uint, configuration string, includeAnyLabelIDs ...uint) *fleet.VPPAppStoreApp {
		tick(f)
		version := &fleet.VPPAppStoreApp{
			AdamID:         "1",
			Platform:       fleet.IOSPlatform,
			VPPAppsTeamsID: versionID,
		}
		if configuration != "" {
			version.Configuration = json.RawMessage(configuration)
			version.ConfigurationUpdatedAt = new(f.now)
		}
		for _, labelID := range includeAnyLabelIDs {
			version.LabelsIncludeAny = append(version.LabelsIncludeAny, fleet.SoftwareScopeLabel{LabelID: labelID})
		}
		f.versionsByFleetID[fleetID] = append(f.versionsByFleetID[fleetID], version)
		return version
	}
	// setConfiguration sets the version's configuration like the API and GitOps do, an empty configuration clears it
	setConfiguration := func(f *testAppStoreAppDatastore, version *fleet.VPPAppStoreApp, configuration string) {
		tick(f)
		version.Configuration = nil
		if configuration != "" {
			version.Configuration = json.RawMessage(configuration)
		}
		version.ConfigurationUpdatedAt = new(f.now)
	}
	// deleteVersion removes the version, and clears it from the installs of it like the version_id foreign key does
	deleteVersion := func(f *testAppStoreAppDatastore, fleetID uint, versionID uint) {
		f.versionsByFleetID[fleetID] = slices.DeleteFunc(f.versionsByFleetID[fleetID], func(version *fleet.VPPAppStoreApp) bool {
			return version.VPPAppsTeamsID == versionID
		})
		for i := range f.installs {
			if ptr.ValOrZero(f.installs[i].VPPAppTeamID) == versionID {
				f.installs[i].VPPAppTeamID = nil
			}
		}
	}
	// install records an install of the version that isn't from the cron, like a manual or setup experience install
	install := func(f *testAppStoreAppDatastore, hostID uint, versionID uint) {
		tick(f)
		f.installs = append(f.installs, fleet.HostVPPSoftwareInstall{HostID: hostID, VPPAppTeamID: new(versionID), CreatedAt: f.now})
	}
	// failInstall records a failed install of the version, like an MDM error or a failed verification
	failInstall := func(f *testAppStoreAppDatastore, hostID uint, versionID uint) {
		tick(f)
		f.failedInstalls = append(f.failedInstalls, fleet.HostVPPSoftwareInstall{HostID: hostID, VPPAppTeamID: new(versionID), CreatedAt: f.now})
	}
	// flagLatestInstall sets the re-send flag on the host's latest install like a change to a Fleet variable the configuration uses
	flagLatestInstall := func(f *testAppStoreAppDatastore, hostID uint) {
		for i := len(f.installs) - 1; i >= 0; i-- {
			if f.installs[i].HostID == hostID {
				f.installs[i].NeedsConfigurationResend = true
				return
			}
		}
	}
	setLabel := func(f *testAppStoreAppDatastore, hostID uint, labelID uint, member bool) {
		if member {
			if f.labelIDsByHostID[hostID] == nil {
				f.labelIDsByHostID[hostID] = map[uint]struct{}{}
			}
			f.labelIDsByHostID[hostID][labelID] = struct{}{}
			return
		}
		delete(f.labelIDsByHostID[hostID], labelID)
	}
	// reconcile runs the cron and returns the version sent to each host in that run
	reconcile := func(f *testAppStoreAppDatastore, ctx context.Context) (map[uint]uint, error) {
		tick(f)
		f.sentVersionByHostID = map[uint]uint{}
		err := ReconcileAppStoreAppConfigurations(ctx, f.ds, f, slog.New(slog.DiscardHandler))
		return f.sentVersionByHostID, err
	}

	t.Run("a host whose version and configuration didn't change gets nothing", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// run the cron twice without changes, the host should get nothing
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("editing or clearing the configuration of a host's version sends that version to the host once", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// edit the configuration, the host should get version 10 once
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// clear the configuration, the host should get version 10 again
		setConfiguration(f, version, "")
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("an install in the same second as a configuration change is sent again", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// change the configuration half a second after the install's second, the install should count as before the change and the host should get version 10
		version.ConfigurationUpdatedAt = new(f.now.Add(500 * time.Millisecond))
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("changing a Fleet variable a host's configuration uses sends the host's version", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict><key>user</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string></dict>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// flag the host's latest install like a SCIM user, group or link change does, the host should get version 10 once
		flagLatestInstall(f, 1)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("adding and removing a host's labels sends the version the host is in scope for", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict><key>v</key><string>a</string></dict>`, 100)
		addVersion(f, 0, 11, `<dict><key>v</key><string>b</string></dict>`)
		addHost(f, 1, 0)
		install(f, 1, 11)

		// add label 100 to the host, as adding it manually, a label spec or a host vitals label cron does, the host should get version 10
		setLabel(f, 1, 100, true)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)

		// remove label 100 from the host, as removing it, replacing or clearing the label's hosts, or re-enrolling does, the host should get version 11
		setLabel(f, 1, 100, false)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)
	})

	t.Run("a host in scope for two versions gets the first-added one, and the label scope query isn't run for a version without labels", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict><key>v</key><string>a</string></dict>`, 100)
		addVersion(f, 0, 11, `<dict><key>v</key><string>b</string></dict>`, 100)
		addVersion(f, 0, 12, `<dict><key>v</key><string>c</string></dict>`)
		addHost(f, 1, 0, 100)
		addHost(f, 2, 0)
		install(f, 1, 12)
		install(f, 2, 11)
		var scopedVersionIDs []uint
		inScope := f.ds.GetIncludedHostIDMapForVPPAppHostsFunc
		f.ds.GetIncludedHostIDMapForVPPAppHostsFunc = func(ctx context.Context, vppAppTeamID uint, hostIDs []uint) (map[uint]struct{}, error) {
			scopedVersionIDs = append(scopedVersionIDs, vppAppTeamID)
			return inScope(ctx, vppAppTeamID, hostIDs)
		}

		// run the cron, the host in label 100 should get version 10 and the other host version 12, without a scope query for version 12
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10, 2: 12}, sent)
		require.Equal(t, []uint{10, 11}, scopedVersionIDs)
	})

	t.Run("excluding a host's label from its version sends the host's next version", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict><key>v</key><string>a</string></dict>`)
		addVersion(f, 0, 11, `<dict><key>v</key><string>b</string></dict>`)
		addHost(f, 1, 0, 200)
		install(f, 1, 10)

		// exclude label 200 from version 10, as a dynamic exclude any label does once the host refetches, the host should get version 11
		version.LabelsExcludeAny = []fleet.SoftwareScopeLabel{{LabelID: 200}}
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)
	})

	t.Run("adding a version a host is in scope for sends it", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`, 100)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// add a version without labels after the host left version 10's label, the host should get version 11
		addVersion(f, 0, 11, `<dict><key>v</key><string>new</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)
	})

	t.Run("deleting a host's version sends the host's next version, with or without a configuration", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`, 100)
		addVersion(f, 0, 11, `<dict/>`, 100)
		addVersion(f, 0, 12, "")
		addHost(f, 1, 0, 100)
		install(f, 1, 10)

		// delete version 10, the host should get version 11
		deleteVersion(f, 0, 10)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)

		// delete version 11, the host should get version 12 without a configuration
		deleteVersion(f, 0, 11)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 12}, sent)
	})

	t.Run("deleting every version of an app or removing it from GitOps sends nothing", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// delete the only version, the host should get nothing since iOS apps aren't removed from hosts
		deleteVersion(f, 0, 10)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("moving a host between versions sends an install only when either version has a configuration", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`, 100)
		addVersion(f, 0, 11, "", 200)
		versionC := addVersion(f, 0, 12, `<dict/>`)
		addHost(f, 1, 0, 100)
		install(f, 1, 10)

		// move the host from version 10 with a configuration to version 11 without one, the host should get version 11
		setLabel(f, 1, 100, false)
		setLabel(f, 1, 200, true)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)

		// clear version 12's configuration after the host's install of version 11, then move the host to it, the host should get nothing
		setConfiguration(f, versionC, "")
		setLabel(f, 1, 200, false)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("moving a host between versions with the same configuration sends nothing, and moving it to a version with another configuration sends that version", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict><key>v</key><string>same</string></dict>`, 100)
		addVersion(f, 0, 11, `<dict><key>v</key><string>same</string></dict>`, 200)
		addVersion(f, 0, 12, `<dict><key>v</key><string>other</string></dict>`)
		addHost(f, 1, 0, 100)
		install(f, 1, 10)

		// move the host from version 10 to version 11 with the same configuration, the host should get nothing
		setLabel(f, 1, 100, false)
		setLabel(f, 1, 200, true)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// move the host to version 12 with another configuration, the host should get version 12
		setLabel(f, 1, 200, false)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 12}, sent)
	})

	t.Run("moving a host between versions with the same configuration sends the new version when a Fleet variable the configuration uses changed", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict><key>user</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string></dict>`, 100)
		addVersion(f, 0, 11, `<dict><key>user</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string></dict>`)
		addHost(f, 1, 0, 100)
		install(f, 1, 10)

		// move the host from version 10 to version 11 with the same configuration and flag its latest install like a SCIM user change does, the host should get version 11 once
		setLabel(f, 1, 100, false)
		flagLatestInstall(f, 1)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("moving a host off a version whose configuration was cleared after the host's install sends the next version", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		versionA := addVersion(f, 0, 10, `<dict/>`, 100)
		addVersion(f, 0, 11, "")
		addHost(f, 1, 0, 100)
		install(f, 1, 10)

		// clear version 10's configuration and move the host to version 11 before the cron runs, the host should get version 11 since it still has version 10's configuration
		setConfiguration(f, versionA, "")
		setLabel(f, 1, 100, false)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)
	})

	t.Run("moving a host between a fleet and Unassigned sends the new fleet's version", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addFleet(f, 5)
		addVersion(f, 0, 10, `<dict><key>v</key><string>unassigned</string></dict>`)
		addVersion(f, 5, 20, `<dict><key>v</key><string>fleet</string></dict>`)
		host := addHost(f, 1, 0)
		install(f, 1, 10)

		// move the host to fleet 5, the host should get version 20
		host.TeamID = new(uint(5))
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 20}, sent)

		// move the host back to Unassigned, as deleting its fleet does, the host should get version 10
		host.TeamID = nil
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("moving a host to another fleet where neither version has a configuration sends an install without one", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addFleet(f, 5)
		addVersion(f, 0, 10, "")
		addVersion(f, 5, 20, "")
		host := addHost(f, 1, 0)
		install(f, 1, 10)

		// move the host to fleet 5, the host should get version 20 since the cron doesn't read other fleets' versions
		host.TeamID = new(uint(5))
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 20}, sent)
	})

	t.Run("a host whose latest install is of a version it isn't in scope for gets the version it's in scope for", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict><key>v</key><string>a</string></dict>`, 100)
		addVersion(f, 0, 11, `<dict><key>v</key><string>b</string></dict>`)
		addHost(f, 1, 0)

		// install version 10 on a host outside its label, as the setup experience or a queued install of a deleted version does, the host should get version 11
		install(f, 1, 10)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 11}, sent)
	})

	t.Run("a host re-created after being deleted gets nothing until Fleet installs the app on it", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 2, 0)

		// run the cron for a host without an install from Fleet, the host should get nothing
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("a host without the app in its inventory gets nothing, and gets the latest configuration if the app is back in its inventory", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// remove the app from the host's inventory and edit the configuration, the host should get nothing
		delete(f.inventoryHostIDs, 1)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// reinstall the app outside Fleet, the host should get version 10
		f.inventoryHostIDs[1] = struct{}{}
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("a host with MDM off gets the configuration once MDM is back on", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// turn MDM off for the host and edit the configuration, the host should get nothing
		f.mdmOffHostIDs[1] = struct{}{}
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// turn MDM back on, the host should get version 10
		delete(f.mdmOffHostIDs, 1)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("an install Apple rejects while the app is out of Apple Business is sent once the app is back", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)

		// remove the app from Apple Business and edit the configuration, the install should be rejected and nothing sent
		f.rejectedHostIDs[1] = struct{}{}
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// add the app back to Apple Business, the host should get version 10
		delete(f.rejectedHostIDs, 1)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("a host with a queued install of its version gets nothing more until the install runs", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		f.queueInstalls = true

		// edit the configuration, the host should get version 10 queued once
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("a host whose install failed within the last hour gets nothing, unless the configuration changes after the failure", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)

		// fail the host's install of the new configuration, as a variable without a value does, the host should get nothing since the install failed less than an hour ago
		failInstall(f, 1, 10)
		f.now = time.Now().Truncate(time.Second)
		f.failedInstalls[0].CreatedAt = f.now
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// edit the configuration again, the host should get version 10 right away since the new configuration might not fail
		setConfiguration(f, version, `<dict><key>a</key><string>c</string></dict>`)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("a host whose install failed more than an hour ago gets its version again", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)

		// fail the host's install two hours ago and run the cron, the host should get version 10 again since the install failed more than an hour ago
		failInstall(f, 1, 10)
		f.failedInstalls[0].CreatedAt = time.Now().Add(-2 * time.Hour)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("a rejected install on one host doesn't stop the next host's install", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		addHost(f, 2, 0)
		install(f, 1, 10)
		install(f, 2, 10)
		f.rejectedHostIDs[1] = struct{}{}

		// edit the configuration with the first host's install rejected, the second host should still get version 10
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{2: 10}, sent)
	})

	t.Run("removing or expiring a fleet's VPP token sends nothing to its hosts", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)

		// expire the fleet's VPP token, the host should get nothing
		f.tokenByFleetID[0].RenewDate = time.Now().Add(-time.Hour)
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)

		// remove the fleet's VPP token, the host should get nothing
		delete(f.tokenByFleetID, 0)
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
	})

	t.Run("an error on one App Store app is logged and the next app is still reconciled", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		versions := f.ds.GetAppStoreAppVersionsByTeamFunc
		f.ds.GetAppStoreAppVersionsByTeamFunc = func(ctx context.Context, fleetID uint) (map[uint][]*fleet.VPPAppStoreApp, error) {
			versionsByTitleID, err := versions(ctx, fleetID)
			versionsByTitleID[2] = []*fleet.VPPAppStoreApp{{VPPAppsTeamsID: 20, LabelsIncludeAny: []fleet.SoftwareScopeLabel{{LabelID: 1}}}}
			return versionsByTitleID, err
		}
		installs := f.ds.ListHostAppStoreAppInstallVersionsFunc
		f.ds.ListHostAppStoreAppInstallVersionsFunc = func(ctx context.Context, fleetID uint, hostIDs []uint) (map[uint]map[uint]fleet.HostVPPSoftwareInstall, error) {
			installByHostIDByTitleID, err := installs(ctx, fleetID, hostIDs)
			installByHostIDByTitleID[2] = installByHostIDByTitleID[1]
			return installByHostIDByTitleID, err
		}
		includedHostIDs := f.ds.GetIncludedHostIDMapForVPPAppHostsFunc
		f.ds.GetIncludedHostIDMapForVPPAppHostsFunc = func(ctx context.Context, vppAppTeamID uint, hostIDs []uint) (map[uint]struct{}, error) {
			if vppAppTeamID == 20 {
				return nil, errors.New("read failed")
			}
			return includedHostIDs(ctx, vppAppTeamID, hostIDs)
		}

		// fail reading the label scope of another app's version, the host should still get version 10
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
	})

	t.Run("hosts are checked in batches that continue from the cursor and start over after the last batch", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		lastHostID := uint(appStoreAppConfigurationReconcileBatchSize + 1)
		for hostID := uint(1); hostID <= lastHostID; hostID++ {
			addHost(f, hostID, 0)
			install(f, hostID, 10)
		}
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)

		// run the cron, the first batch of hosts should get version 10 and the cursor should be set to its last host
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Len(t, sent, appStoreAppConfigurationReconcileBatchSize)
		require.NotContains(t, sent, lastHostID)
		require.Equal(t, lastHostID-1, f.cursor)

		// run the cron again, only the last host should get version 10 and the cursor should be set back to the first host
		sent, err = reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{lastHostID: 10}, sent)
		require.Zero(t, f.cursor)
	})

	t.Run("a fleet whose VPP token or App Store apps can't be read doesn't stop other fleets or the cursor", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addFleet(f, 5)
		addFleet(f, 6)
		version := addVersion(f, 0, 10, `<dict/>`)
		addVersion(f, 5, 20, `<dict/>`)
		addVersion(f, 6, 30, `<dict/>`)
		addHost(f, 1, 0)
		addHost(f, 2, 5)
		addHost(f, 3, 6)
		install(f, 1, 10)
		install(f, 2, 20)
		install(f, 3, 30)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)
		tokens := f.ds.GetVPPTokenByTeamIDFunc
		f.ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
			if ptr.ValOrZero(teamID) == 5 {
				return nil, errors.New("token can't be decrypted")
			}
			return tokens(ctx, teamID)
		}
		versions := f.ds.GetAppStoreAppVersionsByTeamFunc
		f.ds.GetAppStoreAppVersionsByTeamFunc = func(ctx context.Context, fleetID uint) (map[uint][]*fleet.VPPAppStoreApp, error) {
			if fleetID == 6 {
				return nil, errors.New("read failed")
			}
			return versions(ctx, fleetID)
		}

		// fail reading fleet 5's VPP token and fleet 6's App Store apps, the Unassigned host should still get version 10 and the cursor should be saved
		sent, err := reconcile(f, ctx)
		require.NoError(t, err)
		require.Equal(t, map[uint]uint{1: 10}, sent)
		require.True(t, f.ds.SetAppStoreAppConfigurationReconcileCursorFuncInvoked)
	})

	t.Run("a run whose context ended returns the error without saving the cursor", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		endedCtx, cancel := context.WithCancel(ctx)
		cancel()

		// run the cron with a context that already ended, as when the run times out, the run should fail and the cursor shouldn't be saved
		_, err := reconcile(f, endedCtx)
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, f.ds.SetAppStoreAppConfigurationReconcileCursorFuncInvoked)
	})

	t.Run("a Fleet Free instance reconciles nothing", func(t *testing.T) {
		f := newTestAppStoreAppDatastore()
		addFleet(f, 0)
		version := addVersion(f, 0, 10, `<dict/>`)
		addHost(f, 1, 0)
		install(f, 1, 10)
		setConfiguration(f, version, `<dict><key>a</key><string>b</string></dict>`)

		// run the cron on Fleet Free, the host should get nothing and no hosts should be read
		sent, err := reconcile(f, license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierFree}))
		require.NoError(t, err)
		require.Equal(t, nothingSent, sent)
		require.False(t, f.ds.ListIOSAndIPadOSMDMHostsAfterIDFuncInvoked)
	})
}
