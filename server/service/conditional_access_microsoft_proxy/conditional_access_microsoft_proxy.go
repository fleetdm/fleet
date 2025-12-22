// Package conditional_access_microsoft_proxy is the client HTTP package to operate on Entra through Fleet's MS proxy.
package conditional_access_microsoft_proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// Proxy holds functionality to send requests to Entra via Fleet's MS proxy.
type Proxy struct {
	uri          string
	apiKey       string
	originGetter func() (string, error)

	c *http.Client
}

// New creates a Proxy that will use the given URI and API key.
func New(uri string, apiKey string, originGetter func() (string, error)) (*Proxy, error) {
	if _, err := url.Parse(uri); err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}
	return &Proxy{
		uri:    uri,
		apiKey: apiKey,

		originGetter: originGetter,

		c: fleethttp.NewClient(),
	}, nil
}

type createRequest struct {
	TenantID string `json:"entraTenantId"`
}

// CreateResponse returns the tenant ID and the secret of the created integration
// Such credentials are used to authenticate all requests.
type CreateResponse struct {
	TenantID string `json:"entra_tenant_id"`
	Secret   string `json:"fleet_server_secret"`
}

// Create creates the integration on the MS proxy and returns the consent URL.
func (p *Proxy) Create(ctx context.Context, tenantID string) (*CreateResponse, error) {
	var createResponse CreateResponse
	if err := p.post(
		"/api/v1/microsoft-compliance-partner",
		createRequest{TenantID: tenantID},
		&createResponse,
	); err != nil {
		return nil, fmt.Errorf("create integration failed: %w", err)
	}
	return &createResponse, nil
}

// GetResponse holds the settings of the current integration.
type GetResponse struct {
	TenantID        string  `json:"entra_tenant_id"`
	SetupDone       bool    `json:"setup_done"`
	AdminConsentURL string  `json:"admin_consent_url"`
	SetupError      *string `json:"setup_error"`
}

// Get returns the integration settings.
func (p *Proxy) Get(ctx context.Context, tenantID string, secret string) (*GetResponse, error) {
	var getResponse GetResponse
	if err := p.get(
		"/api/v1/microsoft-compliance-partner/settings",
		fmt.Sprintf("entraTenantId=%s&fleetServerSecret=%s", tenantID, secret),
		&getResponse,
	); err != nil {
		return nil, fmt.Errorf("get integration settings failed: %w", err)
	}
	return &getResponse, nil
}

// DeleteResponse contains an error detail if any.
type DeleteResponse struct {
	Error string `json:"error"`
}

// Delete deprovisions the tenant on Microsoft and deletes the integration in the proxy service.
// Returns a fleet.IsNotFound error if the integration doesn't exist.
func (p *Proxy) Delete(ctx context.Context, tenantID string, secret string) (*DeleteResponse, error) {
	var deleteResponse DeleteResponse
	if err := p.delete(
		"/api/v1/microsoft-compliance-partner",
		fmt.Sprintf("entraTenantId=%s&fleetServerSecret=%s", tenantID, secret),
		&deleteResponse,
	); err != nil {
		return nil, fmt.Errorf("delete integration failed: %w", err)
	}
	return &deleteResponse, nil
}

type setComplianceStatusRequest struct {
	TenantID string `json:"entraTenantId"`
	Secret   string `json:"fleetServerSecret"`

	DeviceID          string `json:"deviceId"`
	UserPrincipalName string `json:"userPrincipalName"`

	DeviceManagementState bool   `json:"deviceManagementState"`
	DeviceName            string `json:"deviceName"`
	OS                    string `json:"os"`
	OSVersion             string `json:"osVersion"`
	Compliant             bool   `json:"compliant"`
	LastCheckInTime       int    `json:"lastCheckInTime"`
}

// SetComplianceStatusResponse holds the MessageID to query the status of the "compliance set" operation.
type SetComplianceStatusResponse struct {
	// MessageID holds the ID to use when querying the status of the "compliance set" operation.
	MessageID string `json:"message_id"`
}

// SetComplianceStatus sets the inventory and compliance status of a host.
// Returns the message ID to query the status of the operation (MS has an asynchronous API).
func (p *Proxy) SetComplianceStatus(
	ctx context.Context,
	tenantID string, secret string,
	deviceID string,
	userPrincipalName string,
	mdmEnrolled bool,
	deviceName, osName, osVersion string,
	compliant bool,
	lastCheckInTime time.Time,
) (*SetComplianceStatusResponse, error) {
	var setComplianceStatusResponse SetComplianceStatusResponse
	if err := p.post(
		"/api/v1/microsoft-compliance-partner/device",
		setComplianceStatusRequest{
			TenantID: tenantID,
			Secret:   secret,

			DeviceID:          deviceID,
			UserPrincipalName: userPrincipalName,

			DeviceManagementState: mdmEnrolled,
			DeviceName:            deviceName,
			OS:                    osName,
			OSVersion:             osVersion,
			Compliant:             compliant,
			LastCheckInTime:       int(lastCheckInTime.Unix()),
		},
		&setComplianceStatusResponse,
	); err != nil {
		return nil, fmt.Errorf("set compliance status response failed: %w", err)
	}
	return &setComplianceStatusResponse, nil
}

// MessageStatusCompleted is the value returned when a "compliance set" operation has been successfully applied.
const MessageStatusCompleted = "Completed"

// GetMessageStatusResponse returns the status of a "compliance set" operation.
type GetMessageStatusResponse struct {
	// MessageID is the ID of the operation.
	MessageID string `json:"message_id"`
	// Status of the operation.
	Status string `json:"status"`
	// Detail has some error description when Status is not "Completed".
	Detail *string `json:"detail"`
}

// GetMessageStatus returns the status of the operation (MS has an asynchronous API).
func (p *Proxy) GetMessageStatus(
	ctx context.Context,
	tenantID string, secret string,
	messageID string,
) (*GetMessageStatusResponse, error) {
	var getMessageStatusResponse GetMessageStatusResponse
	if err := p.get(
		"/api/v1/microsoft-compliance-partner/device/message",
		fmt.Sprintf("entraTenantId=%s&fleetServerSecret=%s&messageId=%s", tenantID, secret, messageID),
		&getMessageStatusResponse,
	); err != nil {
		return nil, fmt.Errorf("get message status response failed: %w", err)
	}
	return &getMessageStatusResponse, nil
}

func (p *Proxy) post(path string, request interface{}, response interface{}) error {
	b, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	postRequest, err := http.NewRequest("POST", p.uri+path, nil)
	if err != nil {
		return fmt.Errorf("post create request: %w", err)
	}
	if err := p.setHeaders(postRequest); err != nil {
		return fmt.Errorf("post set headers: %w", err)
	}
	postRequest.Header.Add("Content-Type", "application/json")
	postRequest.Body = io.NopCloser(bytes.NewBuffer(b))
	resp, err := p.c.Do(postRequest)
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("post request failed: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("post read response body: %w", err)
	}
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("post unmarshal response: %w", err)
	}
	return nil
}

func (p *Proxy) get(path string, query string, response interface{}) error {
	getURL := p.uri + path
	if query != "" {
		getURL += "?" + url.PathEscape(query)
	}
	getRequest, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("get create request: %w", err)
	}
	if err := p.setHeaders(getRequest); err != nil {
		return fmt.Errorf("get set headers: %w", err)
	}
	resp, err := p.c.Do(getRequest)
	if err != nil {
		return fmt.Errorf("get request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get request failed: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("get read response body: %w", err)
	}
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("get unmarshal response: %w", err)
	}
	return nil
}

func (p *Proxy) delete(path string, query string, response interface{}) error {
	deleteURL := p.uri + path
	if query != "" {
		deleteURL += "?" + url.PathEscape(query)
	}
	deleteRequest, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("delete create request: %w", err)
	}
	if err := p.setHeaders(deleteRequest); err != nil {
		return fmt.Errorf("delete set headers: %w", err)
	}
	resp, err := p.c.Do(deleteRequest)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusNotFound:
		return &notFoundError{}
	default:
		return fmt.Errorf("delete request failed: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("delete read response body: %w", err)
	}
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("delete unmarshal response: %w", err)
	}
	return nil
}

type notFoundError struct{}

func (e *notFoundError) Error() string {
	return "not found"
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

func (p *Proxy) setHeaders(r *http.Request) error {
	origin, err := p.originGetter()
	if err != nil {
		return fmt.Errorf("get origin: %w", err)
	}
	if origin == "" {
		return fmt.Errorf("missing origin: %w", err)
	}
	r.Header.Add("MS-API-Key", p.apiKey)
	r.Header.Add("Origin", origin)
	return nil
}
