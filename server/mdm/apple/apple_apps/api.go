package apple_apps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type Metadata struct {
	ID         string     `json:"id"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	Name           string                  `json:"name"`
	Platforms      map[string]PlatformData `json:"platformAttributes"`
	DeviceFamilies []string                `json:"deviceFamilies"`
}

type PlatformData struct {
	Artwork           ArtData `json:"artwork"`
	BundleID          string  `json:"bundleId"`
	ExternalVersionID uint    `json:"externalVersionId"`
	LatestVersionInfo LatestVersionInfo
}

func (d PlatformData) IconURL() string {
	// using set values rather than artwork response values for consistency with previous impl
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(d.Artwork.TemplateURL, "{w}", "512"),
			"{h}",
			"512",
		),
		"{f}",
		"png",
	)
}

type LatestVersionInfo struct {
	DisplayVersion string `json:"versionDisplay"`
}

type ArtData struct {
	Height      uint   `json:"height"`
	Width       uint   `json:"width"`
	TemplateURL string `json:"url"`
}

type metadataResp struct {
	Data []Metadata `json:"data"`
}

const appleHostAndScheme = "https://api.ent.apple.com"

// DefaultMetadataRegion is the storefront region used when none is known for an adamID.
// It also heads the fallback chain in GetMetadataWithFallback.
const DefaultMetadataRegion = "us"

// MetadataFallbackRegions is the ordered list of regions to try when an adamID's metadata
// isn't available in the default region. Most App Store apps are global, so fallbacks are
// expected to fire only for region-exclusive apps.
var MetadataFallbackRegions = []string{"us", "gb", "de", "fr", "it", "es"}

// authenticator returns a bearer token for the VPP metadata service (proxied or direct), or an error if once can't be
// retrieved. If forceRenew is true, bypasses the database bearer token cache if it would've otherwise been used.
type authenticator func(forceRenew bool) (string, error)

type Config struct {
	baseURLForRegion func(region string) string
	authenticator    authenticator
}

func StubbedConfig() Config {
	return Config{
		baseURLForRegion: func(region string) string { return getBaseURL(false, region) },
		authenticator:    func(forceRenew bool) (string, error) { return "", nil },
	}
}

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of created as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

// GetMetadata fetches metadata for the given adamIDs from a single storefront region.
// Missing adamIDs are silently omitted from the returned map.
func GetMetadata(adamIDs []string, region string, vppToken string, config Config) (map[string]Metadata, error) {
	if len(adamIDs) == 0 {
		return map[string]Metadata{}, nil
	}
	if region == "" {
		region = DefaultMetadataRegion
	}

	req, err := buildMetadataRequest(config.baseURLForRegion(region), adamIDs, vppToken)
	if err != nil {
		return nil, err
	}

	// small max attempts count because in many cases we're calling this from a UI that does
	// client-side retries on top of this
	var bodyResp metadataResp
	if err = retry.Do(
		func() error { return do(req, config.authenticator, false, &bodyResp) },
		retry.WithInterval(time.Second),
		retry.WithBackoffMultiplier(2),
		retry.WithMaxAttempts(3),
		retry.WithErrorFilter(func(err error) retry.ErrorOutcome {
			// auth retries are handles inside do(); if we get all the way to the outer error,
			// we've already tried to recover and should bail
			if strings.Contains(err.Error(), "auth") {
				return retry.ErrorOutcomeDoNotRetry
			}

			return retry.ErrorOutcomeNormalRetry
		}),
	); err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	metadata := make(map[string]Metadata)
	for _, a := range bodyResp.Data {
		metadata[fmt.Sprint(a.ID)] = a
	}

	return metadata, nil
}

// GetMetadataWithFallback fetches metadata for the given adamIDs, using known regions when
// provided and walking MetadataFallbackRegions for any adamIDs without a known region.
//
// adamIDs:        full set of adamIDs to look up.
// knownRegions:   optional map of adamID -> region to use directly (no fallback). adamIDs
//
//	listed here are grouped by region and fetched in a single batched call
//	per region.
//
// Returns the union of metadata found across all regions plus a map of adamID -> region that
// resolved each one. AdamIDs that couldn't be resolved in any region are silently omitted.
func GetMetadataWithFallback(adamIDs []string, knownRegions map[string]string, vppToken string, config Config) (map[string]Metadata, map[string]string, error) {
	metadata := make(map[string]Metadata, len(adamIDs))
	resolvedRegions := make(map[string]string, len(adamIDs))

	// Group known-region adamIDs by region so we can batch one call per region.
	byRegion := make(map[string][]string)
	var unknowns []string
	for _, id := range adamIDs {
		if region, ok := knownRegions[id]; ok && region != "" {
			byRegion[region] = append(byRegion[region], id)
		} else {
			unknowns = append(unknowns, id)
		}
	}

	for region, ids := range byRegion {
		found, err := GetMetadata(ids, region, vppToken, config)
		if err != nil {
			return nil, nil, err
		}
		for id, m := range found {
			metadata[id] = m
			resolvedRegions[id] = region
		}
	}

	// Walk the fallback chain for adamIDs without a known region.
	remaining := unknowns
	for _, region := range MetadataFallbackRegions {
		if len(remaining) == 0 {
			break
		}
		found, err := GetMetadata(remaining, region, vppToken, config)
		if err != nil {
			return nil, nil, err
		}
		next := remaining[:0]
		for _, id := range remaining {
			if m, ok := found[id]; ok {
				metadata[id] = m
				resolvedRegions[id] = region
			} else {
				next = append(next, id)
			}
		}
		remaining = next
	}

	return metadata, resolvedRegions, nil
}

func buildMetadataRequest(baseURL string, adamIDs []string, vppToken string) (*http.Request, error) {
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base VPP app details URL: %w", err)
	}

	query := reqURL.Query()
	query.Add("ids", strings.Join(adamIDs, ","))
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to VPP app details endpoint: %w", err)
	}
	if strings.HasPrefix(baseURL, appleHostAndScheme) { // Apple requires providing the token as a cookie
		req.Header.Set("Cookie", fmt.Sprintf("itvt=%s", vppToken))
	} else { // Fleet proxy requires providing the token as a header
		req.Header.Set("vpp-token", vppToken)
	}

	return req, nil
}

func do(req *http.Request, getBearerToken authenticator, forceRenew bool, dest *metadataResp) error {
	bearerToken, err := getBearerToken(forceRenew)
	if err != nil {
		return fmt.Errorf("authenticating to VPP app details endpoint: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request to VPP app details endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body from VPP app details endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		limitedBody := body
		if len(limitedBody) > 1000 {
			limitedBody = limitedBody[:1000]
		}

		if resp.StatusCode == http.StatusUnauthorized && !forceRenew {
			return do(req, getBearerToken, true, dest)
		} else if resp.StatusCode >= http.StatusTooManyRequests && resp.Header.Get("Retry-After") != "" {
			retryAfter := resp.Header.Get("Retry-After")
			seconds, err := strconv.ParseInt(retryAfter, 10, 0)
			if err != nil {
				return fmt.Errorf("parsing retry-after header: %w", err)
			}

			ticker := time.NewTicker(time.Duration(seconds) * time.Second)
			defer ticker.Stop()
			<-ticker.C
			return do(req, getBearerToken, false, dest)
		}

		return fmt.Errorf("calling VPP app details endpoint failed with status %d: %s", resp.StatusCode, string(limitedBody))
	}

	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decoding response data from VPP app details endpoint: %w", err)
		}
	}

	return nil
}

func ToVPPApps(app Metadata) map[fleet.InstallableDevicePlatform]fleet.VPPApp {
	// length 1 because watchOS/tvOS/visionOS exist and we don't support them, so using the length of the DeviceFamilies
	// slice would give us extra empty entries
	platforms := make(map[fleet.InstallableDevicePlatform]fleet.VPPApp, 1)
	for _, device := range app.Attributes.DeviceFamilies {
		var (
			data     PlatformData
			ok       bool
			platform fleet.InstallableDevicePlatform
		)

		// It is rare that a single app supports all platforms, but it is possible.
		// Skipping the "appletvos" platform right now as we don't support tvOS;
		// see https://github.com/DIYgod/RSSHub/blob/master/lib/routes/apple/apps.ts for mapping info
		switch device {
		case "iphone":
			data, ok = app.Attributes.Platforms["ios"]
			if !ok {
				continue
			}
			platform = fleet.IOSPlatform
		case "ipad":
			data, ok = app.Attributes.Platforms["ios"]
			if !ok {
				continue
			}
			platform = fleet.IPadOSPlatform
		case "mac":
			data, ok = app.Attributes.Platforms["osx"]
			if !ok {
				continue
			}
			platform = fleet.MacOSPlatform
		default:
			continue
		}

		platforms[platform] = fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   app.ID,
					Platform: platform,
				},
			},
			BundleIdentifier: data.BundleID,
			IconURL:          data.IconURL(),
			Name:             app.Attributes.Name,
			// The "Meta Horizon" app was at some point returned by Apple with `"versionDisplay":" 353.0"`,
			// so we trim the spaces here.
			LatestVersion: strings.TrimSpace(data.LatestVersionInfo.DisplayVersion),
		}
	}
	return platforms
}

func getBaseURL(bearerTokenSupplied bool, region string) string {
	if region == "" {
		region = DefaultMetadataRegion
	}
	if dev_mode.Env("FLEET_DEV_VPP_REGION") != "" {
		// dev override forces a specific region regardless of caller input
		region = dev_mode.Env("FLEET_DEV_VPP_REGION")
	}

	urlFromEnvVar := dev_mode.Env("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL")
	if urlFromEnvVar != "" && urlFromEnvVar != "apple" {
		return dev_mode.Env("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL")
	}
	// if a bearer token is supplied and we don't have an explicit further override, use Apple's endpoint directly
	if urlFromEnvVar == "apple" || bearerTokenSupplied {
		return fmt.Sprintf(appleHostAndScheme+"/v1/catalog/%s/stoken-authenticated-apps?platform=iphone&additionalPlatforms=ipad,mac&extend[apps]=latestVersionInfo", region)
	}

	return fmt.Sprintf("https://fleetdm.com/api/vpp/v1/metadata/%s?platform=iphone&additionalPlatforms=ipad,mac&extend[apps]=latestVersionInfo", region)
}

type authResp struct {
	Token string `json:"fleetServerSecret"`
}

type DataStore interface {
	fleet.GetsAppConfig
	fleet.AccessesMDMConfigAssets
}

func Configure(ctx context.Context, ds DataStore, licenseKey string, token string) Config {
	if token != "" {
		return Config{
			authenticator:    func(forceRenew bool) (string, error) { return token, nil },
			baseURLForRegion: func(region string) string { return getBaseURL(true, region) },
		}
	}

	return Config{
		authenticator:    getAuthenticator(ctx, ds, licenseKey),
		baseURLForRegion: func(region string) string { return getBaseURL(false, region) },
	}
}

func getAuthenticator(ctx context.Context, ds DataStore, licenseKey string) authenticator {
	return func(forceRenew bool) (string, error) {
		const key = fleet.MDMAssetVPPProxyBearerToken
		if !forceRenew {
			// throwing away the error here as on retrieval errors we'll request a new token
			fromDB, _ := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{key}, nil)
			if v, ok := fromDB[key]; ok {
				return string(v.Value), nil
			}
		}

		authUrl := dev_mode.Env("FLEET_DEV_VPP_PROXY_AUTH_URL")
		if authUrl == "" {
			authUrl = "https://fleetdm.com/api/vpp/v1/auth"
		}

		appConfig, err := ds.AppConfig(ctx)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting server URL from app config")
		}

		body, err := json.Marshal(struct {
			ServerURL  string `json:"fleetServerUrl"`
			LicenseKey string `json:"fleetLicenseKey"`
		}{appConfig.ServerSettings.ServerURL, licenseKey})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "encoding authentication request for VPP metadata service")
		}

		req, err := http.NewRequestWithContext(ctx, "POST", authUrl, bytes.NewBuffer(body))
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "building authentication request for VPP metadata service")
		}

		var authResponse authResp
		if err = doAuth(req, &authResponse); err != nil {
			return "", ctxerr.Wrap(ctx, err, "authenticating to VPP metadata service")
		}

		if authResponse.Token == "" {
			return "", ctxerr.New(ctx, "no access token received from VPP metadata service")
		}

		// no need to keep old access tokens around, but no need to hard-fail if we can't clean them up
		_ = ds.HardDeleteMDMConfigAsset(ctx, key)

		// don't fail if we can't persist the token; we can continue anyway and will try again with the next request
		_ = ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{Name: key, Value: []byte(authResponse.Token)})

		return authResponse.Token, nil
	}
}

func doAuth(req *http.Request, dest *authResp) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("authenticating to VPP metadata service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading authentication response from VPP metadata service: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		limitedBody := body
		if len(limitedBody) > 1000 {
			limitedBody = limitedBody[:1000]
		}

		return fmt.Errorf("calling authentication endpoint for VPP metadata service failed with status %d: %s", resp.StatusCode, string(limitedBody))
	}

	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decoding response data from authentication endpoint for VPP metdata service: %w", err)
		}
	}

	return nil
}
