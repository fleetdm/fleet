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
		authz:  authorizer,
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: config.FleetConfig{MDM: config.MDMConfig{AppleConnectJWT: "test-jwt"}},
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
