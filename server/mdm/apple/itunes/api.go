package itunes

import (
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
)

type AssetMetadata struct {
	BundleID   string `json:"bundleId"`
	ArtworkURL string `json:"artworkUrl60"` // TODO(JVE): confirm this is the size we want
	Version    string `json:"version"`
	TrackName  string `json:"trackName"`
	TrackID    uint   `json:"trackId"`
}

type AssetMetadataFilter struct {
	ID     string
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
		addFilter(query, "id", adamIDsParam)
		addFilter(query, "entity", filter.Entity)
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
		metadata[strconv.Itoa(int(a.TrackID))] = a
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

// addFilter adds a filter to the query values if it is not the zero value.
// TODO(JVE): should we move funcs like this into a separate api_utils package? Since this is
// identical to the implementation in vpp.
func addFilter(query url.Values, key string, value any) {
	switch v := value.(type) {
	case string:
		if v != "" {
			query.Add(key, v)
		}
	case *bool:
		if v != nil {
			query.Add(key, strconv.FormatBool(*v))
		}
	case int32:
		if v != 0 {
			query.Add(key, fmt.Sprintf("%d", v))
		}
	}
}
