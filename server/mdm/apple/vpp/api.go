package vpp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/retry"
)

// Asset is a product in the store.
//
// https://developer.apple.com/documentation/devicemanagement/asset
type Asset struct {
	// AdamID is the unique identifier for a product in the store.
	AdamID string `json:"adamId"`
	// PricingParam is the quality of a product in the store.
	// Possible Values are `STDQ` and `PLUS`
	PricingParam string `json:"pricingParam"`
	// AvailableCount is the number of available licenses for this app in the location specified by
	// the VPP token.
	AvailableCount uint `json:"availableCount"`
}

// ErrorResponse represents the response that contains the error that occurs.
//
// https://developer.apple.com/documentation/devicemanagement/errorresponse
type ErrorResponse struct {
	ErrorInfo    ResponseErrorInfo `json:"errorInfo,omitempty"`
	ErrorMessage string            `json:"errorMessage"`
	ErrorNumber  int32             `json:"errorNumber"`
}

// Error implements the Erorrer interface
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("Apple VPP endpoint returned error: %s (error number: %d)", e.ErrorMessage, e.ErrorNumber)
}

// ResponseErrorInfo represents the request-specific information regarding the
// failure.
//
// https://developer.apple.com/documentation/devicemanagement/responseerrorinfo
type ResponseErrorInfo struct {
	Assets        []Asset  `json:"assets"`
	ClientUserIds []string `json:"clientUserIds"`
	SerialNumbers []string `json:"serialNumbers"`
}

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of created as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

// GetConfig fetches the VPP config from Apple's VPP API. This doubles as a
// verification that the user-provided VPP token is valid.
//
// https://developer.apple.com/documentation/devicemanagement/client_config-a40
func GetConfig(token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, getBaseURL()+"/client/config", nil)
	if err != nil {
		return "", fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	var respJSON struct {
		LocationName string `json:"locationName"`
	}

	if err := do(req, token, &respJSON); err != nil {
		return "", fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}

	return respJSON.LocationName, nil
}

// AssociateAssetsRequest is the request for asset management.
type AssociateAssetsRequest struct {
	// Assets are the assets to assign.
	Assets []Asset `json:"assets"`
	// SerialNumbers is the set of identifiers for devices to assign the
	// assets to.
	SerialNumbers []string `json:"serialNumbers"`
}

// AssociateAssets associates assets to serial numbers according the the
// request parameters provided.
//
// https://developer.apple.com/documentation/devicemanagement/associate_assets
func AssociateAssets(token string, params *AssociateAssetsRequest) (string, error) {
	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(params); err != nil {
		return "", fmt.Errorf("encoding params as JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, getBaseURL()+"/assets/associate", &reqBody)
	if err != nil {
		return "", fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	var respBody struct {
		EventID string `json:"eventId"`
	}

	if err := do(req, token, &respBody); err != nil {
		return "", fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}

	return respBody.EventID, nil
}

// AssetFilter represents the filters for querying assets.
type AssetFilter struct {
	// PageIndex is the requested page index.
	PageIndex int32 `json:"pageIndex"`

	// ProductType is the filter for the asset product type.
	// Possible Values: App, Book
	ProductType string `json:"productType"`

	// PricingParam is the filter for the asset product quality.
	// Possible Values: STDQ, PLUS
	PricingParam string `json:"pricingParam"`

	// Revocable is the filter for asset revocability.
	Revocable *bool `json:"revocable"`

	// DeviceAssignable is the filter for asset device assignability.
	DeviceAssignable *bool `json:"deviceAssignable"`

	// MaxAvailableCount is the filter for the maximum inclusive assets available count.
	MaxAvailableCount int32 `json:"maxAvailableCount"`

	// MinAvailableCount is the filter for the minimum inclusive assets available count.
	MinAvailableCount int32 `json:"minAvailableCount"`

	// MaxAssignedCount is the filter for the maximum inclusive assets assigned count.
	MaxAssignedCount int32 `json:"maxAssignedCount"`

	// MinAssignedCount is the filter for the minimum inclusive assets assigned count.
	MinAssignedCount int32 `json:"minAssignedCount"`

	// AdamID is the filter for the asset product unique identifier.
	AdamID string `json:"adamId"`
}

// GetAssets fetches the assets from Apple's VPP API with optional filters.
func GetAssets(token string, filter *AssetFilter) ([]Asset, error) {
	baseURL := getBaseURL() + "/assets"
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	if filter != nil {
		query := url.Values{}
		addFilter(query, "adamId", filter.AdamID)
		addFilter(query, "pricingParam", filter.PricingParam)
		addFilter(query, "productType", filter.ProductType)
		addFilter(query, "revocable", filter.Revocable)
		addFilter(query, "deviceAssignable", filter.DeviceAssignable)
		addFilter(query, "maxAvailableCount", filter.MaxAvailableCount)
		addFilter(query, "minAvailableCount", filter.MinAvailableCount)
		addFilter(query, "maxAssignedCount", filter.MaxAssignedCount)
		addFilter(query, "minAssignedCount", filter.MinAssignedCount)
		addFilter(query, "pageIndex", filter.PageIndex)
		reqURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	var bodyResp struct {
		Assets []Asset `json:"assets"`
	}

	if err = do(req, token, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving assets: %w", err)
	}

	return bodyResp.Assets, nil
}

// AssignmentFilter is a representation of the query params for the Apple "Get Assignments"
// endpoint.
// https://developer.apple.com/documentation/devicemanagement/get_assignments-o3j#query-parameters
type AssignmentFilter struct {
	// The filter for the assignment product's unique identifier.
	AdamID string `json:"adamId"`
	// The filter for the unique identifier of assigned users in your organization.
	ClientUserID string `json:"clientUserId"`
	// The requested page index.
	PageIndex int `json:"pageIndex"`
	// The filter for the unique identifier of assigned devices in your organization.
	SerialNumber string `json:"serialNumber"`
	// The filter for modified assignments since the specified version identifier.
	SinceVersionID string `json:"sinceVersionId"`
}

// Assignment represents an asset assignment for a device.
//
// https://developer.apple.com/documentation/devicemanagement/assignment
type Assignment struct {
	// The unique identifier for a product in the store.
	AdamID string `json:"adamId"`
	// PricingParam is the quality of a product in the store.
	// Possible Values are `STDQ` and `PLUS`
	PricingParam string `json:"pricingParam"`
	// The unique identifier for a device.
	SerialNumber string `json:"serialNumber"`
}

// GetAssignments fetches the assets from Apple's VPP API with optional filters.
//
// https://developer.apple.com/documentation/devicemanagement/get_assignments-o3j
func GetAssignments(token string, filter *AssignmentFilter) ([]Assignment, error) {
	baseURL := getBaseURL() + "/assignments"
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	if filter != nil {
		query := url.Values{}
		addFilter(query, "adamId", filter.AdamID)
		addFilter(query, "clientUserId", filter.ClientUserID)
		addFilter(query, "serialNumber", filter.SerialNumber)
		addFilter(query, "sinceVersionId", filter.SinceVersionID)
		addFilter(query, "pageIndex", filter.PageIndex)
		reqURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	// TODO(roberto): when we get to importing assets assigned by other
	// MDMs we'll need other top-level keys in this struct, and to modify
	// the return value of this function.
	//
	// https://developer.apple.com/documentation/devicemanagement/getassignmentsresponse
	var bodyResp struct {
		Assignments []Assignment `json:"assignments"`
	}

	if err = do(req, token, &bodyResp); err != nil {
		return nil, fmt.Errorf("retrieving assignments: %w", err)
	}

	return bodyResp.Assignments, nil
}

func do[T any](req *http.Request, token string, dest *T) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body from Apple VPP endpoint: %w", err)
	}

	// For HTTP 5xx server error responses, a Retry-After header indicates
	// how long the client must wait before making additional requests.
	//
	// https://developer.apple.com/documentation/devicemanagement/app_and_book_management/handling_error_responses#3742679
	retryAfter := resp.Header.Get("Retry-After")
	if resp.StatusCode == http.StatusInternalServerError && retryAfter != "" {
		seconds, err := strconv.ParseInt(retryAfter, 10, 0)
		if err != nil {
			return fmt.Errorf("parsing retry-after header: %w", err)
		}

		ticker := time.NewTicker(time.Duration(seconds) * time.Second)
		defer ticker.Stop()
		<-ticker.C
		return do(req, token, dest)
	}

	// For some reason, Apple returns 200 OK even if you pass an invalid token in the Auth header.
	// We will need to parse the response and check to see if it contains an error.
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && (errResp.ErrorMessage != "" || errResp.ErrorNumber != 0) {
		switch errResp.ErrorNumber {
		// 9646: There are too many requests for the current
		// Organization and the request has been rejected, either due
		// to high server volume or an MDM issue. Use an
		// incremental/exponential backoff strategy to retry the
		// request until successful.
		//
		// https://developer.apple.com/documentation/devicemanagement/app_and_book_management/handling_error_responses#3783126
		case 9646:
			return retry.Do(
				func() error { return do(req, token, dest) },
				retry.WithBackoffMultiplier(3),
				retry.WithInterval(5*time.Second),
				retry.WithMaxAttempts(3),
			)
		default:
			return &errResp
		}
	}

	if resp.StatusCode != http.StatusOK {
		limitedBody := body
		if len(limitedBody) > 1000 {
			limitedBody = limitedBody[:1000]
		}
		return fmt.Errorf("calling Apple VPP endpoint failed with status %d: %s", resp.StatusCode, string(limitedBody))
	}

	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decoding response data from Apple VPP endpoint: %w", err)
		}
	}

	return nil
}

func getBaseURL() string {
	devURL := os.Getenv("FLEET_DEV_VPP_URL")
	if devURL != "" {
		return devURL
	}
	return "https://vpp.itunes.apple.com/mdm/v2"
}

// addFilter adds a filter to the query values if it is not the zero value.
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
