package vpp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// metadataCall captures one outbound metadata fetch made by the refresh
// loop, so tests can assert on per-region grouping behavior.
type metadataCall struct {
	region  string
	adamIDs []string
	token   string
}

// fakeMetadataServer plays the role of both Fleet's metadata proxy and
// Apple's /assets endpoint for refresh tests. It records each metadata
// request as a metadataCall and yields whatever app payload the test
// configured.
type fakeMetadataServer struct {
	mu       sync.Mutex
	calls    []metadataCall
	apps     map[string]map[string]string // region -> adamID -> name (for distinguishing storefronts)
	versions map[string]string            // adamID -> versionDisplay returned (region-agnostic)
	owned    map[string][]string          // token -> adamIDs that token's GetAssets returns
	failTok  map[string]bool              // tokens whose /assets call returns 500
}

func (f *fakeMetadataServer) handleMetadata(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	region := parts[len(parts)-1]
	q, _ := url.ParseQuery(r.URL.RawQuery)
	ids := strings.Split(q.Get("ids"), ",")
	token := r.Header.Get("vpp-token")

	f.calls = append(f.calls, metadataCall{region: region, adamIDs: append([]string(nil), ids...), token: token})

	type platformData struct {
		BundleID          string                       `json:"bundleId"`
		ExternalVersionID uint                         `json:"externalVersionId"`
		Artwork           apple_apps.ArtData           `json:"artwork"`
		LatestVersionInfo apple_apps.LatestVersionInfo `json:"-"`
		LatestVersionRaw  map[string]string            `json:"latestVersionInfo"`
	}
	type attrs struct {
		Name           string                  `json:"name"`
		Platforms      map[string]platformData `json:"platformAttributes"`
		DeviceFamilies []string                `json:"deviceFamilies"`
	}
	type meta struct {
		ID         string `json:"id"`
		Attributes attrs  `json:"attributes"`
	}
	type resp struct {
		Data []meta `json:"data"`
	}

	out := resp{}
	for _, id := range ids {
		name := id
		if regionApps, ok := f.apps[region]; ok {
			if n, ok := regionApps[id]; ok {
				name = n
			}
		}
		version := f.versions[id]
		if version == "" {
			version = "1.0"
		}
		out.Data = append(out.Data, meta{
			ID: id,
			Attributes: attrs{
				Name:           name,
				DeviceFamilies: []string{"mac"},
				Platforms: map[string]platformData{
					"osx": {
						BundleID: "com.example." + id,
						// Non-empty Artwork URL so apple_apps derives a
						// non-empty IconURL; otherwise the refresh path's
						// empty-metadata guard skips the row.
						Artwork: apple_apps.ArtData{
							TemplateURL: "https://example.test/icon/" + region + "/" + id + ".png",
						},
						LatestVersionRaw: map[string]string{"versionDisplay": version},
					},
				},
			},
		})
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func (f *fakeMetadataServer) handleAssets(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")

	if f.failTok[token] {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorMessage":"transient","errorNumber":500}`))
		return
	}

	type asset struct {
		AdamID         string `json:"adamId"`
		PricingParam   string `json:"pricingParam"`
		AvailableCount uint   `json:"availableCount"`
	}
	type body struct {
		Assets []asset `json:"assets"`
	}
	out := body{}
	for _, adamID := range f.owned[token] {
		out.Assets = append(out.Assets, asset{AdamID: adamID, PricingParam: "STDQ"})
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func newFakeServers(t *testing.T) (metadata *httptest.Server, vppEndpoint *httptest.Server, fake *fakeMetadataServer) {
	fake = &fakeMetadataServer{
		apps:     make(map[string]map[string]string),
		versions: make(map[string]string),
		owned:    make(map[string][]string),
		failTok:  make(map[string]bool),
	}
	metaSrv := httptest.NewServer(http.HandlerFunc(fake.handleMetadata))
	vppSrv := httptest.NewServer(http.HandlerFunc(fake.handleAssets))
	t.Cleanup(metaSrv.Close)
	t.Cleanup(vppSrv.Close)
	return metaSrv, vppSrv, fake
}

func makeRegionConfig(metaSrvURL string) apple_apps.Config {
	return apple_apps.TestConfigWithBaseURLForRegion(func(region string) string {
		return metaSrvURL + "/" + region
	})
}

func TestRefreshVersionsGroupsByCountry(t *testing.T) {
	metaSrv, vppSrv, fake := newFakeServers(t)
	cfg := makeRegionConfig(metaSrv.URL)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	// Set up: 3 apps. Two anchored to "us", one anchored to "de". Two
	// tokens, one US and one DE. Expect exactly one Apple call per
	// country, with the right adamIDs and token.
	fake.versions = map[string]string{"100": "10.0", "200": "20.0", "300": "30.0"}
	fake.owned = map[string][]string{
		"us-token-secret": {"100", "200"},
		"de-token-secret": {"300"},
	}

	apps := []*fleet.VPPApp{
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform}}, Name: "old-name", LatestVersion: "0.1", IconURL: "old", CountryCode: "us"},
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "200", Platform: fleet.MacOSPlatform}}, Name: "old-name", LatestVersion: "0.1", IconURL: "old", CountryCode: "us"},
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "300", Platform: fleet.MacOSPlatform}}, Name: "old-name", LatestVersion: "0.1", IconURL: "old", CountryCode: "de"},
	}
	tokens := []*fleet.VPPTokenDB{
		{ID: 1, OrgName: "us-org", CountryCode: "us", Token: "us-token-secret"},
		{ID: 2, OrgName: "de-org", CountryCode: "de", Token: "de-token-secret"},
	}

	ds := &mock.DataStore{}
	ds.GetAllVPPAppsFunc = func(ctx context.Context) ([]*fleet.VPPApp, error) {
		return apps, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return tokens, nil
	}
	var inserted []*fleet.VPPApp
	ds.InsertVPPAppsFunc = func(ctx context.Context, in []*fleet.VPPApp) error {
		inserted = append(inserted, in...)
		return nil
	}

	require.NoError(t, RefreshVersions(t.Context(), ds, cfg))

	require.Len(t, fake.calls, 2, "expected one metadata call per anchored country")

	regions := map[string]metadataCall{}
	for _, c := range fake.calls {
		regions[c.region] = c
	}
	require.Contains(t, regions, "us")
	require.Contains(t, regions, "de")

	usCall := regions["us"]
	require.Equal(t, "us-token-secret", usCall.token)
	require.ElementsMatch(t, []string{"100", "200"}, usCall.adamIDs)

	deCall := regions["de"]
	require.Equal(t, "de-token-secret", deCall.token)
	require.Equal(t, []string{"300"}, deCall.adamIDs)

	// All three apps had different versions returned, so all three should
	// be in the inserted batch.
	require.Len(t, inserted, 3)
}

func TestRefreshVersionsSkipsAdamIDsWithNoOwningToken(t *testing.T) {
	metaSrv, vppSrv, fake := newFakeServers(t)
	cfg := makeRegionConfig(metaSrv.URL)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	// adamID 999 is anchored to "fr" but no token in Fleet (any country)
	// owns it — should be silently skipped.
	fake.versions = map[string]string{"100": "10.0", "999": "99.0"}
	fake.owned = map[string][]string{"us-token": {"100"}}

	apps := []*fleet.VPPApp{
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform}}, LatestVersion: "0.1", CountryCode: "us"},
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "999", Platform: fleet.MacOSPlatform}}, LatestVersion: "0.1", CountryCode: "fr"},
	}
	tokens := []*fleet.VPPTokenDB{
		{ID: 1, OrgName: "us-org", CountryCode: "us", Token: "us-token"},
	}

	ds := &mock.DataStore{}
	ds.GetAllVPPAppsFunc = func(ctx context.Context) ([]*fleet.VPPApp, error) { return apps, nil }
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	var inserted []*fleet.VPPApp
	ds.InsertVPPAppsFunc = func(ctx context.Context, in []*fleet.VPPApp) error {
		inserted = append(inserted, in...)
		return nil
	}

	require.NoError(t, RefreshVersions(t.Context(), ds, cfg))
	require.Len(t, fake.calls, 1)
	require.Equal(t, "us", fake.calls[0].region)
	require.Equal(t, []string{"100"}, fake.calls[0].adamIDs)
	require.Len(t, inserted, 1)
	require.Equal(t, "100", inserted[0].AdamID)
}

func TestRefreshVersionsReanchorsWhenAnchoredCountryHasNoOwner(t *testing.T) {
	metaSrv, vppSrv, fake := newFakeServers(t)
	cfg := makeRegionConfig(metaSrv.URL)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	// adamID 100 is anchored to "us" but no US token in Fleet owns it (e.g.
	// the original US token was deleted). The DE token owns the app, so the
	// refresh should re-anchor 100 to "de" and fetch metadata from the DE
	// storefront.
	fake.versions = map[string]string{"100": "10.0"}
	fake.apps = map[string]map[string]string{
		"de": {"100": "Todoist DE"},
	}
	fake.owned = map[string][]string{
		"de-token-secret": {"100"},
	}

	apps := []*fleet.VPPApp{
		{
			VPPAppTeam:    fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform}},
			Name:          "Todoist US",
			LatestVersion: "0.1",
			IconURL:       "us-icon",
			CountryCode:   "us",
		},
	}
	tokens := []*fleet.VPPTokenDB{
		{ID: 1, OrgName: "de-org", CountryCode: "de", Token: "de-token-secret"},
	}

	ds := &mock.DataStore{}
	ds.GetAllVPPAppsFunc = func(ctx context.Context) ([]*fleet.VPPApp, error) { return apps, nil }
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	var reanchored []struct {
		adamID  string
		country string
	}
	ds.UpdateVPPAppCountryCodeFunc = func(ctx context.Context, adamID string, _ fleet.InstallableDevicePlatform, country string) error {
		reanchored = append(reanchored, struct {
			adamID  string
			country string
		}{adamID, country})
		return nil
	}
	var inserted []*fleet.VPPApp
	ds.InsertVPPAppsFunc = func(ctx context.Context, in []*fleet.VPPApp) error {
		inserted = append(inserted, in...)
		return nil
	}

	require.NoError(t, RefreshVersions(t.Context(), ds, cfg))

	require.Len(t, fake.calls, 1)
	require.Equal(t, "de", fake.calls[0].region, "metadata fetched from DE store after re-anchor")
	require.Equal(t, "de-token-secret", fake.calls[0].token)
	require.Equal(t, []string{"100"}, fake.calls[0].adamIDs)

	require.Len(t, reanchored, 1)
	require.Equal(t, "100", reanchored[0].adamID)
	require.Equal(t, "de", reanchored[0].country)

	require.Len(t, inserted, 1)
	require.Equal(t, "Todoist DE", inserted[0].Name)
}

func TestRefreshVersionsCustomB2BAppPicksOwningToken(t *testing.T) {
	metaSrv, vppSrv, fake := newFakeServers(t)
	cfg := makeRegionConfig(metaSrv.URL)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	// Two US tokens; the custom B2B app is owned only by the second one.
	// We must end up calling Apple with token-2's secret and the custom
	// app's adamID.
	fake.versions = map[string]string{"500": "5.0"}
	fake.owned = map[string][]string{
		"token-1-secret": nil,
		"token-2-secret": {"500"},
	}

	apps := []*fleet.VPPApp{
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "500", Platform: fleet.MacOSPlatform}}, LatestVersion: "0.0", CountryCode: "us"},
	}
	tokens := []*fleet.VPPTokenDB{
		{ID: 1, OrgName: "alpha", CountryCode: "us", Token: "token-1-secret"},
		{ID: 2, OrgName: "beta", CountryCode: "us", Token: "token-2-secret"},
	}

	ds := &mock.DataStore{}
	ds.GetAllVPPAppsFunc = func(ctx context.Context) ([]*fleet.VPPApp, error) { return apps, nil }
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	ds.InsertVPPAppsFunc = func(ctx context.Context, in []*fleet.VPPApp) error { return nil }

	require.NoError(t, RefreshVersions(t.Context(), ds, cfg))

	require.Len(t, fake.calls, 1)
	require.Equal(t, "us", fake.calls[0].region)
	require.Equal(t, "token-2-secret", fake.calls[0].token)
	require.Equal(t, []string{"500"}, fake.calls[0].adamIDs)
}

// TestRefreshVersionsAnchoredErrorBlocksFallback guards against a
// transient error on the anchored-country token triggering a spurious
// cross-country re-anchor. We can't tell from an error whether that
// token would have owned the app, so the app is silently skipped this
// run and metadata stays stale until the next refresh.
func TestRefreshVersionsAnchoredErrorBlocksFallback(t *testing.T) {
	metaSrv, vppSrv, fake := newFakeServers(t)
	cfg := makeRegionConfig(metaSrv.URL)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	// adamID 100 anchored to "us", US token's GetAssets call errors out.
	// A DE token does own the app, but the anchored-country error must
	// NOT trigger a re-anchor to DE.
	fake.versions = map[string]string{"100": "10.0"}
	fake.owned = map[string][]string{
		"us-token-secret": {"100"},
		"de-token-secret": {"100"},
	}
	fake.failTok = map[string]bool{"us-token-secret": true}

	apps := []*fleet.VPPApp{
		{
			VPPAppTeam:    fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform}},
			Name:          "Anchored App",
			LatestVersion: "0.1",
			IconURL:       "icon",
			CountryCode:   "us",
		},
	}
	tokens := []*fleet.VPPTokenDB{
		{ID: 1, OrgName: "us-org", CountryCode: "us", Token: "us-token-secret"},
		{ID: 2, OrgName: "de-org", CountryCode: "de", Token: "de-token-secret"},
	}

	ds := &mock.DataStore{}
	ds.GetAllVPPAppsFunc = func(ctx context.Context) ([]*fleet.VPPApp, error) { return apps, nil }
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	var reanchored []string
	ds.UpdateVPPAppCountryCodeFunc = func(ctx context.Context, adamID string, _ fleet.InstallableDevicePlatform, country string) error {
		reanchored = append(reanchored, adamID+"->"+country)
		return nil
	}
	ds.InsertVPPAppsFunc = func(ctx context.Context, in []*fleet.VPPApp) error {
		return errors.New("should not be called when only anchored-country lookup errors")
	}

	// RefreshVersions returns nil here even though GetAssets errored, because
	// GetAssets failures are cached and treated as "unknown ownership" rather
	// than propagated as run errors.
	require.NoError(t, RefreshVersions(t.Context(), ds, cfg))

	// No metadata call should have been issued for this app, neither US
	// (errored) nor DE (suppressed by anchoredErrored).
	require.Empty(t, fake.calls, "expected no metadata fetch when anchored-country lookup errored")
	require.Empty(t, reanchored, "must not re-anchor across countries when anchored lookup errored")
}

func TestRefreshVersionsSkipsAppsWithEmptyCountryCode(t *testing.T) {
	metaSrv, vppSrv, fake := newFakeServers(t)
	cfg := makeRegionConfig(metaSrv.URL)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	apps := []*fleet.VPPApp{
		{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "100", Platform: fleet.MacOSPlatform}}, LatestVersion: "0.1", CountryCode: ""},
	}
	tokens := []*fleet.VPPTokenDB{
		{ID: 1, OrgName: "us-org", CountryCode: "us", Token: "us-token"},
	}
	fake.owned = map[string][]string{"us-token": {"100"}}

	ds := &mock.DataStore{}
	ds.GetAllVPPAppsFunc = func(ctx context.Context) ([]*fleet.VPPApp, error) { return apps, nil }
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return tokens, nil }
	ds.InsertVPPAppsFunc = func(ctx context.Context, in []*fleet.VPPApp) error {
		return errors.New("should not be called")
	}

	require.NoError(t, RefreshVersions(t.Context(), ds, cfg))
	require.Empty(t, fake.calls, "should not call Apple for apps without an anchored country")
}
