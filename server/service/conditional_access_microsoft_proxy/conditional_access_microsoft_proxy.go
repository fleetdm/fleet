// Package conditional_access_microsoft_proxy is the client HTTP package to operate on Entra through Fleet's MS proxy.
package conditional_access_microsoft_proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/service/fleetdm_proxy"
)

// Proxy holds functionality to send requests to Entra via Fleet's MS proxy.
type Proxy struct {
	client *fleetdm_proxy.Client
}

// New creates a Proxy that will use the given URI.
func New(uri string, originGetter func() (string, error)) (*Proxy, error) {
	client, err := fleetdm_proxy.New(uri, originGetter, fleethttp.WithNoTimeout())
	if err != nil {
		return nil, err
	}
	return &Proxy{client: client}, nil
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
	if err := p.client.Do(ctx, http.MethodPost,
		"/api/v1/microsoft-compliance-partner", nil,
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
	if err := p.client.Do(ctx, http.MethodGet,
		"/api/v1/microsoft-compliance-partner/settings",
		url.Values{"entraTenantId": {tenantID}, "fleetServerSecret": {secret}},
		nil, &getResponse,
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
	if err := p.client.Do(ctx, http.MethodDelete,
		"/api/v1/microsoft-compliance-partner",
		url.Values{"entraTenantId": {tenantID}, "fleetServerSecret": {secret}},
		nil, &deleteResponse,
	); err != nil {
		return nil, fmt.Errorf("delete integration failed: %w", fleetdm_proxy.NotFoundOn404(err))
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
	if err := p.client.Do(ctx, http.MethodPost,
		"/api/v1/microsoft-compliance-partner/device", nil,
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
	if err := p.client.Do(ctx, http.MethodGet,
		"/api/v1/microsoft-compliance-partner/device/message",
		url.Values{"entraTenantId": {tenantID}, "fleetServerSecret": {secret}, "messageId": {messageID}},
		nil, &getMessageStatusResponse,
	); err != nil {
		return nil, fmt.Errorf("get message status response failed: %w", err)
	}
	return &getMessageStatusResponse, nil
}
