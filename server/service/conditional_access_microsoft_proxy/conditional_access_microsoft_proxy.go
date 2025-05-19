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

type Proxy struct {
	uri          string
	apiKey       string
	originGetter func() (string, error)

	c *http.Client
}

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
type CreateResponse struct {
	TenantID string `json:"entra_tenant_id"`
	Secret   string `json:"fleet_server_secret"`
}

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

type GetResponse struct {
	TenantID        string  `json:"entra_tenant_id"`
	SetupDone       bool    `json:"setup_done"`
	AdminConsented  bool    `json:"admin_consented"`
	AdminConsentURL string  `json:"admin_consent_url"`
	SetupError      *string `json:"setup_error"`
}

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

type DeleteResponse struct {
	Error string `json:"error"`
}

func (p *Proxy) Delete(ctx context.Context, tenantID string, secret string) (*DeleteResponse, error) {
	var deleteResponse DeleteResponse
	if err := p.delete(
		"/api/v1/microsoft-compliance-partner/settings",
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

	DeviceName      string `json:"deviceName"`
	OS              string `json:"os"`
	OSVersion       string `json:"osVersion"`
	Compliant       bool   `json:"compliant"`
	LastCheckInTime int    `json:"lastCheckInTime"`
}
type SetComplianceStatusResponse struct {
	MessageID string `json:"message_id"`
}

func (p *Proxy) SetComplianceStatus(
	ctx context.Context,
	tenantID string, secret string,
	deviceID string,
	userPrincipalName string,
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

			DeviceName:      deviceName,
			OS:              osName,
			OSVersion:       osVersion,
			Compliant:       compliant,
			LastCheckInTime: int(lastCheckInTime.Unix()),
		},
		&setComplianceStatusResponse,
	); err != nil {
		return nil, fmt.Errorf("set compliance status response failed: %w", err)
	}
	return &setComplianceStatusResponse, nil
}

const MessageStatusCompleted = "Completed"

type GetMessageStatusResponse struct {
	MessageID string  `json:"message_id"`
	Status    string  `json:"status"`
	Detail    *string `json:"detail"`
}

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
		deleteURL += "&" + url.PathEscape(query)
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
	if resp.StatusCode != http.StatusOK {
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
