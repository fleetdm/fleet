// Package conditional_access_google_proxy is the HTTP client for fleetdm.com's Google conditional access API.
// fleetdm.com holds Fleet's allowlisted OAuth client; the Fleet server registers with it to get a per-server secret.
package conditional_access_google_proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/str"
)

const integrationPath = "/api/v1/google-conditional-access"

// Proxy sends requests to fleetdm.com's Google conditional access API.
type Proxy struct {
	uri          string
	originGetter func() (string, error)

	c *http.Client
}

// New creates a Proxy that will use the given URI.
func New(uri string, originGetter func() (string, error)) (*Proxy, error) {
	if _, err := url.Parse(uri); err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}
	return &Proxy{
		uri:          uri,
		originGetter: originGetter,
		c:            fleethttp.NewClient(),
	}, nil
}

type createRequest struct {
	CustomerID string `json:"googleCustomerId"`
}

// CreateResponse holds the credentials of the registration on fleetdm.com.
type CreateResponse struct {
	// FleetServerSecret authenticates this Fleet server on later calls, in both directions.
	FleetServerSecret string `json:"fleet_server_secret"`
	// OAuthClientID is Fleet's Web OAuth client ID, which the Android app uses to sign in.
	OAuthClientID string `json:"oauth_client_id"`
}

// Create registers the Fleet server (identified by the Origin header) for the given Google customer ID.
func (p *Proxy) Create(ctx context.Context, customerID string) (*CreateResponse, error) {
	b, err := json.Marshal(createRequest{CustomerID: customerID})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var createResponse CreateResponse
	if err := p.do(ctx, http.MethodPost, p.uri+integrationPath, bytes.NewReader(b), &createResponse); err != nil {
		return nil, fmt.Errorf("create google conditional access registration: %w", err)
	}
	if createResponse.FleetServerSecret == "" || createResponse.OAuthClientID == "" {
		return nil, errors.New("create google conditional access registration: response is missing fleet_server_secret or oauth_client_id")
	}
	return &createResponse, nil
}

// Delete deletes the registration. Returns a fleet.IsNotFound error if fleetdm.com has no such registration.
func (p *Proxy) Delete(ctx context.Context, customerID, secret string) error {
	query := url.Values{"googleCustomerId": {customerID}, "fleetServerSecret": {secret}}
	if err := p.do(ctx, http.MethodDelete, p.uri+integrationPath+"?"+query.Encode(), nil, nil); err != nil {
		return fmt.Errorf("delete google conditional access registration: %w", err)
	}
	return nil
}

func (p *Proxy) do(ctx context.Context, method, reqURL string, body io.Reader, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	origin, err := p.originGetter()
	if err != nil {
		return fmt.Errorf("get origin: %w", err)
	}
	if origin == "" {
		return errors.New("missing origin: the Fleet server URL is not set")
	}
	req.Header.Set("Origin", origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.c.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return &notFoundError{}
	case resp.StatusCode != http.StatusOK:
		return newStatusError(resp)
	}
	if response == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

type notFoundError struct{}

func (e *notFoundError) Error() string       { return "not found" }
func (e *notFoundError) IsNotFound() bool    { return true }
func (e *notFoundError) IsClientError() bool { return true }

// StatusError is returned for a non-200 response. It carries the HTTP status
// code and response body so callers can surface the remote error.
type StatusError struct {
	Code    int
	RespErr string
}

func (e *StatusError) Error() string {
	if e.RespErr == "" {
		return fmt.Sprintf("%d", e.Code)
	}
	return fmt.Sprintf("%d: %s", e.Code, e.RespErr)
}

// StatusCode returns the remote HTTP status code.
func (e *StatusError) StatusCode() int { return e.Code }

// Body returns the remote response body.
func (e *StatusError) Body() string { return e.RespErr }

func newStatusError(resp *http.Response) *StatusError {
	lr := io.LimitReader(resp.Body, str.MaxErrorResponseBytes+1)
	bodyBytes, _ := io.ReadAll(lr)
	return &StatusError{Code: resp.StatusCode, RespErr: str.TruncateErrorResponse(string(bodyBytes))}
}
