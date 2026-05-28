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

// IsUnknownClientUserError reports whether err indicates that Apple does not
// recognize the clientUserId(s) Fleet sent on an associate-assets or
// assignment query — typically because the user record was retired or never
// completed registration on Apple's side, while Fleet still has it cached as
// 'registered'.
//
// Used by the install flow to self-heal: on this error the caller should
// re-register the VPP user via the v1 endpoint, replace the stale row, and
// retry the original associate-assets call once.
//
// Confirmed code from production traffic:
//   - 9609 / "Unable to find the registered user."
//
// Other codes (9605, 9612, 9627) are listed defensively against Apple's
// docs; same substring backstop as IsMaxDevicesPerUserError catches future
// drift.
func IsUnknownClientUserError(err error) bool {
	if err == nil {
		return false
	}
	var resp *ErrorResponse
	if !errors.As(err, &resp) || resp == nil {
		return false
	}
	switch resp.ErrorNumber {
	case 9605, 9609, 9612, 9627:
		return true
	}
	msg := strings.ToLower(resp.ErrorMessage)
	if strings.Contains(msg, "unable to find") &&
		(strings.Contains(msg, "registered user") || strings.Contains(msg, "client user") || strings.Contains(msg, "user")) {
		return true
	}
	if strings.Contains(msg, "client user") &&
		(strings.Contains(msg, "not found") || strings.Contains(msg, "unknown")) {
		return true
	}
	if strings.Contains(msg, "user not found") {
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
	if r == nil {
		return errors.New("AssociateAssetsRequest: params cannot be nil")
	}
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

// DisassociateAssetsRequest is the body for Apple's disassociate-assets
// endpoint. The shape is identical to AssociateAssetsRequest (same assets +
// mutually-exclusive serialNumbers/clientUserIds contract), so it shares the
// type and Validate.
type DisassociateAssetsRequest = AssociateAssetsRequest

// DisassociateAssets releases (un-reserves) assets previously associated to
// serial numbers or client user IDs, returning the reserved seats to the
// VPP location. Use this to avoid leaking a license when an install that
// reserved a seat does not proceed (e.g. the activation failed or was
// cancelled before reaching the device).
//
// https://developer.apple.com/documentation/devicemanagement/disassociate_assets
func DisassociateAssets(token string, params *DisassociateAssetsRequest) (string, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}

	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(params); err != nil {
		return "", fmt.Errorf("encoding params as JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, getBaseURL()+"/assets/disassociate", &reqBody)
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

// RegisterUserResponse mirrors Apple's synchronous v1 register-user response.
//
// Apple's v1 API responds synchronously with the full user record on success.
// Status is 0 on success and -1 on error; the optional Error* fields carry
// the application-level failure reason. UserID is Apple's assigned identifier.
//
// https://developer.apple.com/documentation/devicemanagement/registervppuserresponse
type RegisterUserResponse struct {
	Status       int    `json:"status"`
	ErrorNumber  int32  `json:"errorNumber,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	User         *struct {
		UserID            json.Number `json:"userId"`
		Status            string      `json:"status"`
		ClientUserIDStr   string      `json:"clientUserIdStr"`
		ManagedAppleIDStr string      `json:"managedAppleIDStr"`
		Email             string      `json:"email,omitempty"`
		InviteURL         string      `json:"inviteUrl,omitempty"`
		InviteCode        string      `json:"inviteCode,omitempty"`
	} `json:"user,omitempty"`
}

// RegisterUser registers a single VPP user via Apple's legacy synchronous v1
// register-user endpoint. Use this rather than the v2 /users/create flow when
// the caller needs definitive confirmation that registration succeeded before
// proceeding — v1 returns the full user record (with Apple's assigned userId)
// in the same response, whereas v2 only returns an eventId that must be
// polled separately.
//
// On any Apple-level failure (status != 0 or non-2xx with error payload),
// returns an *ErrorResponse so callers can distinguish known error codes
// (e.g. invalid Managed Apple ID).
//
// https://developer.apple.com/documentation/devicemanagement/registervppuserrequest
func RegisterUser(token, clientUserID, managedAppleID string) (string, error) {
	if clientUserID == "" || managedAppleID == "" {
		return "", errors.New("RegisterUser: clientUserId and managedAppleId are required")
	}

	// v1 takes the VPP server token in the body rather than the
	// Authorization header.
	reqParams := struct {
		SToken            string `json:"sToken"`
		ClientUserIDStr   string `json:"clientUserIdStr"`
		ManagedAppleIDStr string `json:"managedAppleIDStr"`
	}{
		SToken:            token,
		ClientUserIDStr:   clientUserID,
		ManagedAppleIDStr: managedAppleID,
	}

	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(reqParams); err != nil {
		return "", fmt.Errorf("encoding params as JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, getV1BaseURL()+"/registerVPPUserSrv", &reqBody)
	if err != nil {
		return "", fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	var resp RegisterUserResponse
	if err := do(req, "", &resp); err != nil {
		return "", fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}

	// v1 reports application-level failures via status == -1 with an
	// errorMessage/errorNumber but a 200 transport. Surface those through the
	// same *ErrorResponse callers already handle for v2.
	if resp.Status != 0 || resp.ErrorNumber != 0 || resp.ErrorMessage != "" {
		return "", &ErrorResponse{
			ErrorMessage: resp.ErrorMessage,
			ErrorNumber:  resp.ErrorNumber,
		}
	}
	if resp.User == nil || resp.User.UserID == "" {
		return "", errors.New("Apple VPP register-user returned no user record on success")
	}

	return resp.User.UserID.String(), nil
}

// VPPUserStatus mirrors the lifecycle states Apple reports in v2 /users
// responses for a VPP user.
type VPPUserStatus string

const (
	// VPPUserStatusRegistered: Apple has accepted the registration and
	// issued an invite, but the end user has not yet linked their Apple
	// Account.
	VPPUserStatusRegistered VPPUserStatus = "Registered"
	// VPPUserStatusAssociated: end user has accepted the invite and the
	// Apple Account is bound to the VPP user record.
	VPPUserStatusAssociated VPPUserStatus = "Associated"
	// VPPUserStatusRetired: the user has been retired; a new registration
	// for the same Managed Apple ID is permitted at this location.
	VPPUserStatusRetired VPPUserStatus = "Retired"
)

// User is a single entry from Apple's v2 /users list response.
type User struct {
	ClientUserID string        `json:"clientUserId"`
	IDHash       string        `json:"idHash"`
	Status       VPPUserStatus `json:"status"`
}

// GetUserByManagedAppleID looks up the active VPP user for the given Managed
// Apple ID at the location identified by the bearer token. Apple enforces
// uniqueness on (location, managedAppleId), so a successful response carries
// at most one non-retired user.
//
// Returns (nil, nil) when Apple has no user (or only retired users) for the
// Apple ID — callers should fall through to RegisterUser in that case.
// Returns a non-nil error only for transport / Apple-application errors.
//
// Used by the install self-heal path to recover a stale clientUserId after
// Fleet's local cache drifts from Apple's record (e.g. a stale DB restore).
//
// https://developer.apple.com/documentation/devicemanagement/get-users
func GetUserByManagedAppleID(ctx context.Context, token, managedAppleID string) (*User, error) {
	if managedAppleID == "" {
		return nil, errors.New("GetUserByManagedAppleID: managedAppleID is required")
	}

	q := url.Values{}
	q.Set("managedAppleId", managedAppleID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getBaseURL()+"/users?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to Apple VPP endpoint: %w", err)
	}

	var resp struct {
		Users []User `json:"users"`
	}
	if err := do(req, token, &resp); err != nil {
		return nil, fmt.Errorf("making request to Apple VPP endpoint: %w", err)
	}

	// Apple's contract is at-most-one non-retired user per (location, Apple ID),
	// but we scan the full slice defensively in case a Retired ghost is
	// returned alongside an active record on some iOS revision.
	for i := range resp.Users {
		if resp.Users[i].Status != VPPUserStatusRetired {
			return &resp.Users[i], nil
		}
	}
	return nil, nil
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
	// v1 endpoints carry the token in the request body, so callers pass an
	// empty string to skip the Authorization header.
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

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

// getV1BaseURL returns the base URL for Apple's legacy v1 VPP endpoints. The
// dev override (FLEET_DEV_VPP_URL) is returned as-is so tests can mock both
// v1 and v2 against the same httptest server using path-based routing.
func getV1BaseURL() string {
	devURL := dev_mode.Env("FLEET_DEV_VPP_URL")
	if devURL != "" {
		return devURL
	}
	return "https://vpp.itunes.apple.com/mdm"
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
