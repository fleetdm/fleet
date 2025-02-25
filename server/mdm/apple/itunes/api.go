package itunes

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

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of created as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

func GetAssetMetadata(adamIDs []string, filter *AssetMetadataFilter) (map[string]AssetMetadata, error) {
	baseURL := getBaseURL()
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base iTunes URL: %w", err)
	}

	adamIDsParam := strings.Join(adamIDs, ",")

	if filter != nil {
		query := url.Values{}
		query.Add("id", adamIDsParam)
		query.Add("entity", filter.Entity)
		reqURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to Apple iTunes endpoint: %w", err)
	}

	var bodyResp struct {
		Results []AssetMetadata `json:"results"`
	}

	if err = do(req, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving asset metadata: %w", err)
	}

	metadata := make(map[string]AssetMetadata)
	for _, a := range bodyResp.Results {
		metadata[fmt.Sprint(a.TrackID)] = a
	}

	return metadata, nil
}

func do[T any](req *http.Request, dest *T) error {
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
				func() error { return do(req, dest) },
				retry.WithInterval(1*time.Second),
				retry.WithMaxAttempts(4),
			)

		}

		return fmt.Errorf("calling Apple iTunes endpoint failed with status %d: %s", resp.StatusCode, string(limitedBody))
	}

	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decoding response data from Apple iTunes endpoint: %w", err)
		}
	}

	return nil
}

func getBaseURL() string {
	devURL := os.Getenv("FLEET_DEV_ITUNES_URL")
	if devURL != "" {
		return devURL
	}
	return "https://itunes.apple.com/lookup"
}
