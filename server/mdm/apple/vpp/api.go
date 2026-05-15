package vpp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
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
	// AvailableCount is the number of available licenses for this app in the organization unit
	// specified by the VPP token.
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

// IsMaxDevicesPerUserError reports whether err is an Apple VPP error indicating
// that a Managed Apple ID has reached the per-user device cap (Apple allows
// up to 5 devices per user license).
//
// Apple's numeric code for this case has not been stable across iOS releases,
// so the helper matches by message substring as well. Confirm against Apple's
// sandbox before locking in the canonical code.
func IsMaxDevicesPerUserError(err error) bool {
	if err == nil {
		return false
	}
	var resp *ErrorResponse
	if !errors.As(err, &resp) || resp == nil {
		return false
	}
	// Known candidate code — refine against sandbox results.
	if resp.ErrorNumber == 9622 {
		return true
	}
	msg := strings.ToLower(resp.ErrorMessage)
	if strings.Contains(msg, "maximum number of devices") ||
		strings.Contains(msg, "device limit") {
		return true
	}
	return false
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

// ClientConfig is the subset of Apple's /client/config response that Fleet
// uses. CountryCode is the lowercase ISO 3166-1 alpha-2 code (e.g. "us",
// "de") of the storefront associated with the token.
type ClientConfig struct {
	LocationName string
	CountryCode  string
}

// GetConfig fetches the VPP config from Apple's VPP API. This doubles as a
// verification that the user-provided VPP token is valid. The call is wrapped
// in a 3-attempt retry; the returned country code is lowercased. ctx bounds
// the entire retry sequence so callers can cap user-facing latency with a
// deadline.
//
// https://developer.apple.com/documentation/devicemanagement/client_config-a40
func GetConfig(ctx context.Context, token string) (ClientConfig, error) {
	var cfg ClientConfig
	var returnErr error

	_ = retry.Do(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, getBaseURL()+"/client/config", nil)
		if err != nil {
			returnErr = ctxerr.Wrap(ctx, err, "creating request to Apple VPP endpoint")
			// don't retry on request construction errors
			return nil
		}

		// Apple's /client/config response uses countryISO2ACode for the ISO
		// 3166-1 alpha-2 storefront country (e.g. "US", "DE"). Verified
		// empirically — the developer docs aren't loaded here.
		var respJSON struct {
			LocationName     string `json:"locationName"`
			CountryISO2ACode string `json:"countryISO2ACode"`
		}

		if err := do(req, token, &respJSON); err != nil {
			returnErr = ctxerr.Wrap(ctx, err, "making request to Apple VPP endpoint")

			// Don't retry on Apple application errors (e.g. invalid token);
			// only on transient transport-level failures.
			var appleErr *ErrorResponse
			if errors.As(err, &appleErr) {
				return nil
			}

			// retry on other errors
			return err
		}

		cfg = ClientConfig{
			LocationName: respJSON.LocationName,
			CountryCode:  strings.ToLower(respJSON.CountryISO2ACode),
		}
		returnErr = nil
		return nil
	},
		retry.WithBackoffMultiplier(2),
		retry.WithInterval(500*time.Millisecond),
		retry.WithMaxAttempts(3),
	)

	return cfg, returnErr
}

// AssociateAssetsRequest is the request for asset management.
//
// Apple accepts EITHER SerialNumbers OR ClientUserIds, never both — see Validate.
type AssociateAssetsRequest struct {
	// Assets are the assets to assign.
	Assets []Asset `json:"assets"`
	// SerialNumbers is the set of identifiers for devices to assign the
	// assets to. Used for device-scoped licensing on manually-enrolled and
	// DEP-enrolled hosts.
	SerialNumbers []string `json:"serialNumbers,omitempty"`
	// ClientUserIds is the set of Fleet-generated identifiers for VPP users
	// (registered via CreateUsers) to assign the assets to. Used for
	// user-scoped licensing on Account-Driven User Enrolled (BYOD) hosts.
	ClientUserIds []string `json:"clientUserIds,omitempty"`
}

// Validate enforces Apple's contract that exactly one of SerialNumbers or
// ClientUserIds is set on an associate-assets request.
func (r *AssociateAssetsRequest) Validate() error {
	hasSerials := len(r.SerialNumbers) > 0
	hasUsers := len(r.ClientUserIds) > 0
	switch {
	case hasSerials && hasUsers:
		return errors.New("AssociateAssetsRequest: SerialNumbers and ClientUserIds are mutually exclusive")
	case !hasSerials && !hasUsers:
		return errors.New("AssociateAssetsRequest: one of SerialNumbers or ClientUserIds is required")
	}
	return nil
}

// AssociateAssets associates assets to serial numbers or client user IDs
// according the the request parameters provided.
//
// https://developer.apple.com/documentation/devicemanagement/associate_assets
func AssociateAssets(token string, params *AssociateAssetsRequest) (string, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}

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

// CreateUsersRequest is the body for Apple's create-users endpoint.
//
// https://developer.apple.com/documentation/devicemanagement/create-users
type CreateUsersRequest struct {
	Users []CreateUsersUser `json:"users"`
}

// CreateUsersUser identifies a single VPP user to register against an Apple VPP
// location. ClientUserId is a stable, Fleet-generated UUID; ManagedAppleId is
// the user's Managed Apple ID surfaced from the host's TokenUpdate.
type CreateUsersUser struct {
	ClientUserId   string `json:"clientUserId"`
	ManagedAppleId string `json:"managedAppleId"`
}

// CreateUsersResponse is the body returned by Apple's create-users endpoint.
//
// On success, EventID is populated and Users echoes back the registrations,
// each carrying Apple's assigned UserId. Apple may return per-user errors
// (e.g., for one of several requested users); callers must inspect each
// CreateUsersResult.ErrorMessage / ErrorNumber to distinguish partial
// failures from a fully successful batch.
type CreateUsersResponse struct {
	EventID string              `json:"eventId"`
	Users   []CreateUsersResult `json:"users"`
}

// CreateUsersResult mirrors the per-user fields Apple may return.
//
// Modeled defensively: only ClientUserId is guaranteed in the response. UserId
// is Apple's assigned identifier on success; the optional Error* fields carry
// per-user partial-failure information.
type CreateUsersResult struct {
	UserId         string `json:"userId,omitempty"`
	ClientUserId   string `json:"clientUserId"`
	ManagedAppleId string `json:"managedAppleId,omitempty"`
	Status         string `json:"status,omitempty"`
	InviteCode     string `json:"inviteCode,omitempty"`
	InviteURL      string `json:"inviteUrl,omitempty"`
	// ErrorMessage and ErrorNumber are populated when Apple rejects this
	// individual user even though the overall request succeeded with 200.
	ErrorMessage string `json:"errorMessage,omitempty"`
	ErrorNumber  int32  `json:"errorNumber,omitempty"`
}

// HasError returns true if Apple flagged this individual user as failed.
func (r *CreateUsersResult) HasError() bool {
	return r.ErrorNumber != 0 || r.ErrorMessage != ""
}

// CreateUsers registers VPP users at Apple's create-users endpoint and returns
// Apple's response. A nil error indicates the request itself succeeded; per-user
// partial failures are surfaced on each CreateUsersResult — callers must check
// HasError() on each entry rather than relying solely on the function's error.
//
// https://developer.apple.com/documentation/devicemanagement/create-users
func CreateUsers(token string, params *CreateUsersRequest) (*CreateUsersResponse, error) {
	if params == nil || len(params.Users) == 0 {
		return nil, errors.New("CreateUsersRequest: at least one user is required")
	}

	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(params); err != nil {
		return nil, fmt.Errorf("encoding params as JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, getBaseURL()+"/users/create", &reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	var respBody CreateUsersResponse
	if err := do(req, token, &respBody); err != nil {
		return nil, fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}

	return &respBody, nil
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
func GetAssets(ctx context.Context, token string, filter *AssetFilter) ([]Asset, error) {
	var assets []Asset
	var returnErr error

	_ = retry.Do(func() error {
		var err error
		assets, err = getAssetsOnce(ctx, token, filter)
		returnErr = err

		var ne net.Error
		// if we still have some time left on the current request's context
		// deadline and the error is a timeout, we may retry
		if dl, _ := ctx.Deadline(); (dl.IsZero() || time.Until(dl) >= time.Second) && errors.As(err, &ne) && ne.Timeout() {
			// will retry
			return err
		}
		// returnErr may be != nil, but it's not an error that we should retry
		return nil
	},
		retry.WithBackoffMultiplier(3),
		retry.WithInterval(100*time.Millisecond),
		retry.WithMaxAttempts(3),
	)
	return assets, returnErr
}

func getAssetsOnce(ctx context.Context, token string, filter *AssetFilter) ([]Asset, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Reset the request body for retries. After client.Do reads the body,
	// it's consumed. GetBody (set by http.NewRequest for *bytes.Buffer)
	// returns a fresh reader over the original bytes.
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return fmt.Errorf("resetting request body for VPP retry: %w", err)
		}
		req.Body = body
	}

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
	devURL := dev_mode.Env("FLEET_DEV_VPP_URL")
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
