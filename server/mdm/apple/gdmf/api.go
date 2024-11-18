package gdmf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const baseURL = "https://gdmf.apple.com/v2/pmv"

// Asset represents the metadata for an asset in the Apple Software Lookup Service[1][2].
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
// [2]:
// https://support.apple.com/guide/deployment/use-mdm-to-deploy-software-updates-depafd2fad80/web
type Asset struct {
	ProductVersion   string   `json:"ProductVersion"`
	Build            string   `json:"Build"`
	PostingDate      string   `json:"PostingDate"`
	ExpirationDate   string   `json:"ExpirationDate"`
	SupportedDevices []string `json:"SupportedDevices"`
}

// AssetSets represents the metadata for a set of assets in the Apple Software Lookup Service[1][2].
// [1]: http://gdmf.apple.com/v2/pmv
// [2]: https://support.apple.com/guide/deployment/use-mdm-to-deploy-software-updates-depafd2fad80/web
type AssetSets struct {
	IOS   []Asset `json:"iOS"`
	MacOS []Asset `json:"macOS"`
	// VisionOS []Asset `json:"visionOS"` // Fleet doesn't support visionOS yet
	// XROS     []Asset `json:"xrOS"`    // Fleet doesn't support xrOS yet
}

// APIResponse represents the response from the Apple Software Lookup Service[1][2].
// [1]: http://gdmf.apple.com/v2/pmv
// [2]: https://support.apple.com/guide/deployment/use-mdm-to-deploy-software-updates-depafd2fad80/web
type APIResponse struct {
	PublicAssetSets AssetSets `json:"PublicAssetSets"`
	AssetSets       AssetSets `json:"AssetSets"`
	// PublicRapidSecurityResponses interface{} `json:"PublicRapidSecurityResponses"` // Fleet doesn't support PublicRapidSecurityResponses yet
}

// GetLatestOSVersion returns the latest OS version for the given device. The device is matched
// against the Apple Software Update Lookup Service[1][2] to find the latest version. If no matching
// asset is found, an error is returned.
// [1]: http://gdmf.apple.com/v2/pmv
// [2]: https://support.apple.com/guide/deployment/use-mdm-to-deploy-software-updates-depafd2fad80/web
func GetLatestOSVersion(device fleet.MDMAppleMachineInfo) (*Asset, error) {
	r, err := GetAssetMetadata()
	if err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	assetSet := r.PublicAssetSets.MacOS // default to public asset set; note that if the device is not macOS, iPhone, or iPad, we'll fail to match the supported device and return an error below
	if strings.HasPrefix(device.Product, "iPhone") ||
		strings.HasPrefix(device.Product, "iPad") ||
		strings.HasPrefix(device.SoftwareUpdateDeviceID, "iPhone") ||
		strings.HasPrefix(device.SoftwareUpdateDeviceID, "iPad") {
		assetSet = r.PublicAssetSets.IOS
	}
	latestIdx := -1
	for i, s := range assetSet {
		for _, d := range s.SupportedDevices {
			if d == device.Product || d == device.SoftwareUpdateDeviceID {
				if latestIdx == -1 {
					latestIdx = i // first match found, update the index
					continue
				}
				if fleet.CompareVersions(assetSet[latestIdx].ProductVersion, s.ProductVersion) < 0 {
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

// GetAssetMetadata retrieves the asset metadata from the Apple Software Lookup Service[1][2].
// [1]: http://gdmf.apple.com/v2/pmv
// [2]: https://support.apple.com/guide/deployment/use-mdm-to-deploy-software-updates-depafd2fad80/web
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

	resp, err := doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body from Apple endpoint: %w", err)
	}
	var dest APIResponse
	if err := json.Unmarshal(body, &dest); err != nil {
		return nil, fmt.Errorf("decoding response data from Apple endpoint: %w", err)
	}

	return &dest, nil
}

func doWithRetry(req *http.Request) (*http.Response, error) {
	const (
		maxRetries           = 3
		retryBackoff         = 1 * time.Second
		maxWaitForRetryAfter = 10 * time.Second
	)
	var resp *http.Response
	var err error
	op := func() error {
		resp, err = client.Do(req)
		if err != nil {
			return err
		}

		defer func() {
			if resp != nil && resp.StatusCode >= http.StatusBadRequest {
				// consume and close the body for retried requests to prevent resource leaks
				_, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}()

		if resp.StatusCode == http.StatusTooManyRequests {
			// handle 429 rate-limits
			rawAfter := resp.Header.Get("Retry-After")
			afterSecs, err := strconv.ParseInt(rawAfter, 10, 0)
			if err == nil && (time.Duration(afterSecs)*time.Second) < maxWaitForRetryAfter {
				// the retry-after duration is reasonable, wait for it and return a
				// retryable error so that we try again.
				time.Sleep(time.Duration(afterSecs) * time.Second)
				return errors.New("retry after requested delay")
			}
		}
		if resp.StatusCode >= http.StatusBadRequest {
			// 400+ status can be worth retrying
			return fmt.Errorf("calling gdmf endpoint failed with status %d", resp.StatusCode)
		}
		return nil
	}

	if err := backoff.Retry(op, backoff.WithMaxRetries(backoff.NewConstantBackOff(retryBackoff), uint64(maxRetries))); err != nil {
		return nil, err
	}

	return resp, err
}

func getBaseURL() string {
	devURL := os.Getenv("FLEET_DEV_GDMF_URL")
	if devURL != "" {
		return devURL
	}
	return baseURL
}
