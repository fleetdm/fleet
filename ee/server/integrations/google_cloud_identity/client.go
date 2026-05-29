package google_cloud_identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Default endpoints. Overridable in tests via NewClient's opts.
const (
	defaultCloudIdentityBase = "https://cloudidentity.googleapis.com"
	defaultDirectoryBase     = "https://admin.googleapis.com/admin/directory/v1"
	defaultHTTPTimeout       = 20 * time.Second
)

// ComplianceState enum values per Cloud Identity ClientState schema.
type ComplianceState string

const (
	ComplianceStateUnspecified  ComplianceState = "COMPLIANCE_STATE_UNSPECIFIED"
	ComplianceStateCompliant    ComplianceState = "COMPLIANT"
	ComplianceStateNonCompliant ComplianceState = "NON_COMPLIANT"
)

// ManagedState enum values.
type ManagedState string

const (
	ManagedStateUnspecified ManagedState = "MANAGED_STATE_UNSPECIFIED"
	ManagedStateManaged     ManagedState = "MANAGED"
	ManagedStateUnmanaged   ManagedState = "UNMANAGED"
)

// HealthScore enum values.
type HealthScore string

const (
	HealthScoreUnspecified HealthScore = "HEALTH_SCORE_UNSPECIFIED"
	HealthScoreVeryPoor    HealthScore = "VERY_POOR"
	HealthScorePoor        HealthScore = "POOR"
	HealthScoreNeutral     HealthScore = "NEUTRAL"
	HealthScoreGood        HealthScore = "GOOD"
	HealthScoreVeryGood    HealthScore = "VERY_GOOD"
)

// ClientState is the subset of the Cloud Identity ClientState resource Fleet
// reads and writes. Per the schema, every PATCHable field is here; output-only
// fields (createTime, lastUpdateTime, ownerType) are loaded but not written.
type ClientState struct {
	// Name is the full resource name:
	// "devices/{deviceId}/deviceUsers/{deviceUserId}/clientState/{partner}".
	// Partner = "{customerID-without-C}-{suffix}" in non-Alliance mode.
	Name string `json:"name,omitempty"`
	// CustomID — caller-supplied stable identifier (Fleet uses host.uuid).
	CustomID string `json:"customId,omitempty"`
	// AssetTags — caller-supplied tags (Fleet uses team name + label set).
	AssetTags []string `json:"assetTags,omitempty"`
	// HealthScore enum.
	HealthScore HealthScore `json:"healthScore,omitempty"`
	// ScoreReason — human-readable explanation for the health score.
	ScoreReason string `json:"scoreReason,omitempty"`
	// Managed enum.
	Managed ManagedState `json:"managed,omitempty"`
	// ComplianceState enum.
	ComplianceState ComplianceState `json:"complianceState,omitempty"`
	// KeyValuePairs — partner-specific extension map. Values are custom
	// attribute unions {numberValue|stringValue|boolValue}.
	KeyValuePairs map[string]CustomAttributeValue `json:"keyValuePairs,omitempty"`
	// Etag — required for optimistic concurrency on subsequent PATCHes.
	Etag string `json:"etag,omitempty"`

	// Output-only.
	CreateTime     string `json:"createTime,omitempty"`
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	OwnerType      string `json:"ownerType,omitempty"`
}

// CustomAttributeValue is a union; exactly one of the three fields should be
// set per entry.
type CustomAttributeValue struct {
	NumberValue *float64 `json:"numberValue,omitempty"`
	StringValue *string  `json:"stringValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

// DeviceUserLookupResponse is the response from devices.deviceUsers.lookup.
type DeviceUserLookupResponse struct {
	// Names contains the matched deviceUser resource names.
	Names []string `json:"names,omitempty"`
}

// Customer is the partial response from admin.directory.customers/my_customer.
type Customer struct {
	ID            string `json:"id"`
	CustomerDomain string `json:"customerDomain,omitempty"`
}

// Operation is the long-running operation response shape used by approve/block
// and some PATCHes. The PATCH on ClientState is synchronous; the long-running
// shape is documented but not observed for partner ClientStates in practice.
type Operation struct {
	Name     string          `json:"name,omitempty"`
	Done     bool            `json:"done,omitempty"`
	Error    *operationError `json:"error,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

type operationError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Client is a thin HTTP wrapper around the Cloud Identity v1beta1 and
// Directory v1 endpoints Fleet uses.
type Client struct {
	httpClient        *http.Client
	cloudIdentityBase string
	directoryBase     string
}

// ClientOption configures a Client at construction.
type ClientOption func(*Client)

// WithHTTPClient overrides the default http.Client (mainly for tests).
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cl *Client) { cl.httpClient = c }
}

// WithCloudIdentityBase overrides the cloudidentity.googleapis.com base URL
// (used by tests to point at an httptest server).
func WithCloudIdentityBase(base string) ClientOption {
	return func(cl *Client) { cl.cloudIdentityBase = strings.TrimRight(base, "/") }
}

// WithDirectoryBase overrides the admin.googleapis.com directory base URL.
func WithDirectoryBase(base string) ClientOption {
	return func(cl *Client) { cl.directoryBase = strings.TrimRight(base, "/") }
}

// NewClient constructs a Cloud Identity REST client backed by the given
// OAuth2 token source.
func NewClient(ctx context.Context, tokenSource oauth2.TokenSource, opts ...ClientOption) *Client {
	hc := oauth2.NewClient(ctx, tokenSource)
	hc.Timeout = defaultHTTPTimeout

	c := &Client{
		httpClient:        hc,
		cloudIdentityBase: defaultCloudIdentityBase,
		directoryBase:     defaultDirectoryBase,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetCustomer calls admin.googleapis.com/admin/directory/v1/customers/my_customer
// and returns the customer ID. The integration's startup verifies the
// returned ID matches the configured customer_id.
func (c *Client) GetCustomer(ctx context.Context) (*Customer, error) {
	u := c.directoryBase + "/customers/my_customer"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build customer request: %w", err)
	}
	var customer Customer
	if err := c.do(req, &customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

// LookupDeviceUserByRawResourceID resolves an Endpoint Verification
// device_resource_id to the canonical
// "devices/{deviceId}/deviceUsers/{deviceUserId}" name.
//
// Per the API reference, this is the unambiguous resolution path when the
// caller has read the EV-local accounts.json: one resource ID maps to one
// deviceUser, no shared-device ambiguity.
func (c *Client) LookupDeviceUserByRawResourceID(ctx context.Context, rawResourceID string) (*DeviceUserLookupResponse, error) {
	// The lookup is a GET on a wildcard parent — `devices/-` matches any
	// device the caller has access to under the customer.
	u := fmt.Sprintf(
		"%s/v1beta1/devices/-/deviceUsers:lookup?rawResourceId=%s",
		c.cloudIdentityBase,
		url.QueryEscape(rawResourceID),
	)
	return c.doLookup(ctx, u)
}

// LookupDeviceUserByEmail resolves a Workspace email to every deviceUser
// resource currently associated with it. This is the fallback path for hosts
// without EV; on shared devices the response will include deviceUsers from
// every device the user has ever signed in on, so the caller must
// post-filter.
func (c *Client) LookupDeviceUserByEmail(ctx context.Context, email string) (*DeviceUserLookupResponse, error) {
	u := fmt.Sprintf(
		"%s/v1beta1/devices/-/deviceUsers:lookup?userId=%s",
		c.cloudIdentityBase,
		url.QueryEscape(email),
	)
	return c.doLookup(ctx, u)
}

func (c *Client) doLookup(ctx context.Context, u string) (*DeviceUserLookupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build lookup request: %w", err)
	}
	var resp DeviceUserLookupResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PatchClientStateRequest is the input to PatchClientState.
type PatchClientStateRequest struct {
	// DeviceUserResource is "devices/{deviceId}/deviceUsers/{deviceUserId}".
	DeviceUserResource string
	// Partner is the partner segment of the ClientState name:
	// "{customerID-without-C}-{suffix}" in non-Alliance mode.
	Partner string
	// Customer is required and identifies the customer owning the data.
	// Format: "customers/{customerId}".
	Customer string
	// State is the ClientState body to PATCH.
	State *ClientState
	// UpdateMask is the comma-joined list of field names to patch. Recommended:
	// "complianceState,managed,healthScore,scoreReason,customId,assetTags,keyValuePairs"
	// for a full update.
	UpdateMask string
}

// PatchClientState writes Fleet's desired ClientState to Cloud Identity. The
// returned ClientState contains the etag the next PATCH should send.
func (c *Client) PatchClientState(ctx context.Context, in PatchClientStateRequest) (*ClientState, error) {
	if in.DeviceUserResource == "" {
		return nil, errors.New("PatchClientState: DeviceUserResource is required")
	}
	if in.Partner == "" {
		return nil, errors.New("PatchClientState: Partner is required")
	}
	if in.Customer == "" {
		return nil, errors.New("PatchClientState: Customer is required")
	}
	if in.State == nil {
		return nil, errors.New("PatchClientState: State is required")
	}

	resourceName := fmt.Sprintf("%s/clientState/%s", in.DeviceUserResource, in.Partner)
	u := fmt.Sprintf(
		"%s/v1beta1/%s?customer=%s&updateMask=%s",
		c.cloudIdentityBase,
		resourceName,
		url.QueryEscape(in.Customer),
		url.QueryEscape(in.UpdateMask),
	)

	body, err := json.Marshal(in.State)
	if err != nil {
		return nil, fmt.Errorf("marshal ClientState: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build PATCH request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp ClientState
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// APIError represents a non-2xx response from Cloud Identity / Directory APIs.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("google_cloud_identity: HTTP %d: %s", e.StatusCode, e.Body)
}

// IsPermissionDenied reports whether the error came back as 403, which on
// Cloud Identity ClientState PATCH most commonly means the customer's
// Workspace edition does not include Cloud Identity Premium security
// management (see proposal Customer-side prerequisites section).
func IsPermissionDenied(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusForbidden
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
