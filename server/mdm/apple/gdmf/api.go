package gdmf

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
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"golang.org/x/mod/semver"
)

const baseURL = "https://gdmf.apple.com/v2/pmv"

type AssetMetadata struct {
	BundleID         string   `json:"bundleId"`
	ArtworkURL       string   `json:"artworkUrl512"`
	Version          string   `json:"version"`
	TrackName        string   `json:"trackName"`
	TrackID          uint     `json:"trackId"`
	SupportedDevices []string `json:"supportedDevices"`
}

type AssetMetadataFilter struct {
	Entity string
}

// Asset represents the metadata for an asset in the Apple Software Update API[1].
// Example:
//
//	{
//	    "ProductVersion": "14.6.1",
//	    "Build": "23G93",
//	    "PostingDate": "2024-08-07",
//	    "ExpirationDate": "2024-11-11",
//	    "SupportedDevices": [
//	        "J132AP",
//	        "VMA2MACOSAP",
//	        "VMM-x86_64"
//	    ]
//	}
//
// [1]: http://gdmf.apple.com/v2/pmv
type Asset struct {
	ProductVersion   string   `json:"ProductVersion"`
	Build            string   `json:"Build"`
	PostingDate      string   `json:"PostingDate"`
	ExpirationDate   string   `json:"ExpirationDate"`
	SupportedDevices []string `json:"SupportedDevices"`
}

// AssetSets represents the metadata for a set of assets in the Apple Software Update API[1].
// [1]: http://gdmf.apple.com/v2/pmv
type AssetSets struct {
	IOS   []Asset `json:"iOS"`
	MacOS []Asset `json:"macOS"`
	// VisionOS []Asset `json:"visionOS"` // Fleet doesn't support visionOS yet
	// XROS     []Asset `json:"xrOS"`    // Fleet doesn't support xrOS yet
}

// APIResponse represents the response from the Apple Software Update API[1].
// [1]: http://gdmf.apple.com/v2/pmv
type APIResponse struct {
	PublicAssetSets AssetSets `json:"PublicAssetSets"`
	AssetSets       AssetSets `json:"AssetSets"`
	// PublicRapidSecurityResponses interface{} `json:"PublicRapidSecurityResponses"` // Fleet doesn't support PublicRapidSecurityResponses yet
}

func GetLatestOSVersion(device apple_mdm.MachineInfo) (*Asset, error) {
	r, err := GetAssetMetadata()
	if err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	assetSet := r.PublicAssetSets.MacOS // Default to public asset set; note that if the device is not macOS, iPhone, or iPad, we'll fail to match the supported device and return an error below
	if strings.HasPrefix(device.Product, "iPhone") || strings.HasPrefix(device.Product, "iPad") {
		assetSet = r.PublicAssetSets.IOS
	}
	latestIdx := -1
	for i, a := range assetSet {
		for _, sd := range a.SupportedDevices {
			if sd == device.Product || sd == device.SoftwareUpdateDeviceID {
				if latestIdx == -1 {
					latestIdx = i // first supported device found, update the index
					continue
				}
				if semver.Compare(a.ProductVersion, assetSet[latestIdx].ProductVersion) > 0 {
					latestIdx = i // found a later version, update the index
				}
			}
		}
	}
	if latestIdx == -1 {
		return nil, fmt.Errorf("no matching asset found for device %s", device.Product)
	}
	return &assetSet[latestIdx], nil
}

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of created as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

func GetAssetMetadata() (*APIResponse, error) {
	baseURL := getBaseURL()
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to Apple endpoint: %w", err)
	}
	req.Header.Set("User-Agent", "fleet-device-management")

	var bodyResp APIResponse

	if err = do(req, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	return &bodyResp, nil
}

func do[T any](req *http.Request, dest *T) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request to Apple endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body from Apple endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// limitedBody := body
		// if len(limitedBody) > 1000 {
		// 	limitedBody = limitedBody[:1000]
		// }

		if resp.StatusCode >= http.StatusInternalServerError {
			return retry.Do(
				func() error { return do(req, dest) },
				retry.WithInterval(1*time.Second),
				retry.WithMaxAttempts(4),
			)
		}

		return fmt.Errorf("calling Apple endpoint failed with status %d: %s", resp.StatusCode, string(body))
	}

	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decoding response data from Apple endpoint: %w", err)
		}
	}

	return nil
}

func getBaseURL() string {
	devURL := os.Getenv("FLEET_DEV_GDMF_URL")
	if devURL != "" {
		return devURL
	}
	return baseURL
}
