package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
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
			_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
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
			_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
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

	t.Run("Fails for Fleet Agent Android apps via GitOps", func(t *testing.T) {
		ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
			return nil, nil
		}

		fleetAgentPackages := []string{
			"com.fleetdm.agent",
			"com.fleetdm.agent.pingali",
			"com.fleetdm.agent.private.testuser",
		}

		for _, pkg := range fleetAgentPackages {
			t.Run(pkg+" dry run", func(t *testing.T) {
				_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
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
				_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
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
