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
	ExternalVersionID string  `json:"externalVersionId"`
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

	query := url.Values{}
	query.Add("ids", strings.Join(adamIDs, ","))
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to VPP app details endpoint: %w", err)
	}

	var bodyResp struct {
		Data []Metadata `json:"data"`
	}

	if err = do(req, vppToken, bearerToken, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	metadata := make(map[string]Metadata)
	for _, a := range bodyResp.Data {
		metadata[fmt.Sprint(a.ID)] = a
	}

	return metadata, nil
}

func do[T any](req *http.Request, vppToken string, bearerToken string, dest *T) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Add("Cookie", fmt.Sprintf("itvt=%s", vppToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request to Apple iTunes endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body from Apple iTunes endpoint: %w", err)
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

func getBaseURL() string {
	// Use https://api.ent.apple.com/v1/catalog/us/stoken-authenticated-apps?platform=iphone,ipad,mac&extend[apps]=latestVersionInfo to access Apple directly
	devURL := os.Getenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL")
	if devURL != "" {
		return devURL
	}
	return "https://fleetdm.com/api/vpp/v1/metadata/us?platform=iphone&additionalPlatforms=ipad,mac&extend[apps]=latestVersionInfo"
}
