package apple_apps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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

type Authenticator func(forceRenew bool) (string, error)

func (a Authenticator) WithForcedRenew() Authenticator {
	return func(bool) (string, error) { return a(true) }
}

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of created as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

func GetMetadata(adamIDs []string, vppToken string, getBearerToken Authenticator) (map[string]Metadata, error) {
	baseURL := getBaseURL()
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

	var bodyResp metadataResp
	if err = do(req, vppToken, getBearerToken, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	metadata := make(map[string]Metadata)
	for _, a := range bodyResp.Data {
		metadata[fmt.Sprint(a.ID)] = a
	}

	return metadata, nil
}

func do(req *http.Request, vppToken string, getBearerToken Authenticator, dest *metadataResp) error {
	bearerToken, err := getBearerToken(false)
	if err != nil {
		return retry.Do(
			func() error { return do(req, vppToken, getBearerToken, dest) },
			retry.WithMaxAttempts(1),
		)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Add("Cookie", fmt.Sprintf("itvt=%s", vppToken))

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

		if resp.StatusCode == http.StatusUnauthorized {
			return retry.Do(
				func() error {
					return do(req, vppToken, getBearerToken.WithForcedRenew(), dest)
				},
				retry.WithMaxAttempts(1),
			)
		} else if resp.StatusCode >= http.StatusTooManyRequests && resp.Header.Get("Retry-After") != "" {
			retryAfter := resp.Header.Get("Retry-After")
			if resp.StatusCode == http.StatusInternalServerError && retryAfter != "" {
				seconds, err := strconv.ParseInt(retryAfter, 10, 0)
				if err != nil {
					return fmt.Errorf("parsing retry-after header: %w", err)
				}

				ticker := time.NewTicker(time.Duration(seconds) * time.Second)
				defer ticker.Stop()
				<-ticker.C
				return do(req, vppToken, getBearerToken, dest)
			}
		} else if resp.StatusCode >= http.StatusInternalServerError {
			return retry.Do(
				func() error { return do(req, vppToken, getBearerToken.WithForcedRenew(), dest) },
				retry.WithInterval(time.Second),
				retry.WithBackoffMultiplier(2),
				retry.WithMaxAttempts(4),
			)
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
			LatestVersion:    data.LatestVersionInfo.DisplayVersion,
		}
	}
	return platforms
}

func getBaseURL() string {
	devURL := os.Getenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL")
	if devURL == "apple" {
		return "https://api.ent.apple.com/v1/catalog/us/stoken-authenticated-apps?platform=iphone&additionalPlatforms=ipad,mac&extend[apps]=latestVersionInfo"
	}
	if devURL != "" {
		return devURL
	}
	return "https://fleetdm.com/api/vpp/v1/metadata/us?platform=iphone&additionalPlatforms=ipad,mac&extend[apps]=latestVersionInfo"
}

type authResp struct {
	Token string `json:"fleetServerSecret"`
}

func GetAuthenticator(ctx context.Context, ds fleet.AccessesMDMConfigAssets, licenseKey string, myServerURL string) Authenticator {
	token := os.Getenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN")
	if token != "" {
		return func(bool) (string, error) { return token, nil }
	}

	return func(forceRenew bool) (string, error) {
		const key = fleet.MDMAssetVPPProxyBearerToken
		if !forceRenew {
			// throwing away the error here as on retrieval errors we'll request a new token
			fromDB, _ := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{key}, nil)
			if v, ok := fromDB[key]; ok {
				return string(v.Value), nil
			}
		}

		authUrl := os.Getenv("FLEET_DEV_VPP_PROXY_AUTH_URL")
		if authUrl == "" {
			authUrl = "https://fleetdm.com/api/vpp/v1/auth"
		}

		body, err := json.Marshal(struct {
			ServerURL string `json:"serverUrl"`
		}{myServerURL})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "encoding authentication request for VPP metadata service")
		}

		req, err := http.NewRequestWithContext(ctx, "POST", authUrl, bytes.NewBuffer(body))
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "building authentication request for VPP metadata service")
		}

		var authResponse authResp
		if err = doAuth(req, licenseKey, &authResponse); err != nil {
			return "", ctxerr.Wrap(ctx, err, "authenticating to VPP metadata service")
		}

		if authResponse.Token == "" {
			return "", ctxerr.New(ctx, "no access token received from VPP metadata service")
		}

		// don't fail if we can't persist the token; we can continue anyway and will try again with the next request
		_ = ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{Name: key, Value: []byte(authResponse.Token)})

		return authResponse.Token, nil
	}
}

func doAuth(req *http.Request, licenseKey string, dest *authResp) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", licenseKey))

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
