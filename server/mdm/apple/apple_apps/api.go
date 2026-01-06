package apple_apps

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/retry"
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

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of created as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

func GetMetadata(adamIDs []string, vppToken string, bearerToken string) (map[string]Metadata, error) {
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
	if err = do(req, vppToken, bearerToken, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	metadata := make(map[string]Metadata)
	for _, a := range bodyResp.Data {
		metadata[fmt.Sprint(a.ID)] = a
	}

	return metadata, nil
}

func do(req *http.Request, vppToken string, bearerToken string, dest *metadataResp) error {
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

		if resp.StatusCode >= http.StatusInternalServerError {
			return retry.Do(
				func() error { return do(req, vppToken, bearerToken, dest) },
				retry.WithInterval(1*time.Second),
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

func GetAppMetadataBearerToken(ds any) string {
	token := os.Getenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN")
	if token != "" {
		return token
	}

	return "" // this will fail downstream
}
