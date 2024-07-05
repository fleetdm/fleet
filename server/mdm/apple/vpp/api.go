package vpp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// Asset is a product in the store.
type Asset struct {
	// AdamID is the unique identifier for a product in the store.
	AdamID string `json:"adamId"`
	// PricingParam is the quality of a product in the store.
	// Possible Values are `STDQ` and `PLUS`
	PricingParam string `json:"pricingParam"`
}

// ErrorResponse represents the response that contains the error that occurs.
type ErrorResponse struct {
	ErrorInfo    ResponseErrorInfo `json:"errorInfo"`
	ErrorMessage string            `json:"errorMessage"`
	ErrorNumber  int32             `json:"errorNumber"`
}

// ResponseErrorInfo represents the request-specific information regarding the
// failure.
type ResponseErrorInfo struct {
	Assets        []Asset  `json:"assets"`
	ClientUserIds []string `json:"clientUserIds"`
	SerialNumbers []string `json:"serialNumbers"`
}

// client is a package-level client (similar to http.DefaultClient) so it can
// be reused instead of crated as needed, as the internal Transport typically
// has internal state (cached connections, etc) and it's safe for concurrent
// use.
var client = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

// GetConfig fetches the VPP config from Apple's VPP API. This doubles as a
// verification that the user-provided VPP token is valid.
func GetConfig(token string) (string, bool, error) {
	req, err := http.NewRequest(http.MethodGet, getBaseURL()+"/client/config", nil)
	if err != nil {
		return "", false, fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	resp, err := do(req, token)
	if err != nil {
		return "", false, fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("reading response body from Apple VPP endpoint: %w", err)
	}

	// For some reason, Apple returns 200 OK even if you pass an invalid token in the Auth header.
	// We will need to parse the response and check to see if it contains an error.
	var respJSON struct {
		LocationName string `json:"locationName"`
		ErrorNumber  int    `json:"errorNumber"`
	}

	if err := json.Unmarshal(body, &respJSON); err != nil {
		return "", false, fmt.Errorf("parsing response body from Apple VPP endpoint: %w", err)
	}

	// Per https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
	if resp.StatusCode == 401 || respJSON.ErrorNumber == 9622 {
		return "", false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("calling Apple VPP config endpoint failed with status %d", resp.StatusCode)
	}

	return respJSON.LocationName, true, nil
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
func AssociateAssets(token string, params *AssociateAssetsRequest) error {
	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(params); err != nil {
		return fmt.Errorf("encoding params as JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, getBaseURL()+"/assets/associate", &reqBody)
	if err != nil {
		return fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	resp, err := do(req, token)
	if err != nil {
		return fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("decoding error response from Apple VPP endpoint: %w", err)
		}
		return fmt.Errorf("Apple VPP endpoint returned error: %s (error number: %d)", errResp.ErrorMessage, errResp.ErrorNumber)
	}

	return nil
}

func do(req *http.Request, token string) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	return client.Do(req)
}

func getBaseURL() string {
	devURL := os.Getenv("FLEET_DEV_VPP_URL")
	if devURL != "" {
		return devURL
	}
	return "https://vpp.itunes.apple.com/mdm/v2"
}
