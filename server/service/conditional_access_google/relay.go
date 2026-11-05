package conditional_access_google

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

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/str"
)

const registrationPath = "/api/v1/google-conditional-access"

// maxRelayResponseBytes bounds a successful relay response body.
const maxRelayResponseBytes = 1 << 20

// relay sends JSON requests to the Google conditional access relay on fleetdm.com, with
// this server's URL in the Origin header, which the relay uses to call the server back.
// RelayOption configures NewRelay.
type RelayOption func(*relayOptions)

type relayOptions struct {
	allowHTTP bool
}

// AllowHTTP accepts a plain HTTP relay URI, for a mock relay in development and tests.
// Without it only HTTPS is accepted, since the relay exchanges the server secret.
func AllowHTTP() RelayOption {
	return func(o *relayOptions) { o.allowHTTP = true }
}

type relay struct {
	uri          string
	uriErr       error
	originGetter func() (string, error)
	c            *http.Client
}

// NewRelay creates the relay client. uri is the fleetdm.com service and originGetter
// returns this Fleet server's URL, which fleetdm.com uses to call the server back. A uri
// that parses but can't be used as a base fails each request instead, so an unused
// integration with a bad URI doesn't stop the server from starting.
func NewRelay(uri string, originGetter func() (string, error), opts ...RelayOption) (Relay, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse relay uri: %w", err)
	}
	var o relayOptions
	for _, opt := range opts {
		opt(&o)
	}
	var uriErr error
	switch {
	case u.Scheme == "" || u.Host == "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "":
		uriErr = fmt.Errorf("invalid relay uri %q: a scheme and host are required, with no query or fragment", uri)
	case u.Scheme != "https" && (u.Scheme != "http" || !o.allowHTTP):
		uriErr = fmt.Errorf("invalid relay uri %q: must use https", uri)
	}
	// Every call runs inside an admin's request, so it keeps the default timeout. Redirects
	// aren't followed: a 307 or 308 would resend the server secret to another host.
	return &relay{
		uri:          strings.TrimRight(uri, "/"),
		uriErr:       uriErr,
		originGetter: originGetter,
		c:            fleethttp.NewClient(fleethttp.WithFollowRedir(false)),
	}, nil
}

type registerRequest struct {
	CustomerID string `json:"googleCustomerId"`
}

// Registration holds the per-server secret and Fleet's Web OAuth client ID.
type Registration struct {
	FleetServerSecret string `json:"fleetServerSecret"`
	// TODO: the Android app is its only consumer and may not need it; drop it if so.
	OAuthClientID string `json:"oauthClientId"`
}

func (r *relay) Register(ctx context.Context, customerID string) (*Registration, error) {
	var resp Registration
	if err := r.send(ctx, http.MethodPost, registerRequest{CustomerID: customerID}, &resp); err != nil {
		return nil, fmt.Errorf("register with fleetdm.com: %w", err)
	}
	if resp.FleetServerSecret == "" {
		return nil, errors.New("register with fleetdm.com: empty fleetServerSecret in response")
	}
	return &resp, nil
}

type deregisterRequest struct {
	CustomerID string `json:"googleCustomerId"`
	Secret     string `json:"fleetServerSecret"`
}

func (r *relay) Deregister(ctx context.Context, customerID, secret string) error {
	err := r.send(ctx, http.MethodDelete, deregisterRequest{CustomerID: customerID, Secret: secret}, nil)
	if se, ok := errors.AsType[*RelayStatusError](err); ok && se.Code == http.StatusNotFound {
		err = &relayNotFoundError{RelayStatusError: se}
	}
	if err != nil {
		return fmt.Errorf("deregister from fleetdm.com: %w", err)
	}
	return nil
}

// send sends body as JSON to the registration path and decodes a 2xx JSON response into
// response (nil to ignore it). An empty 2xx body is only accepted when response is nil.
// Any other status returns a *RelayStatusError.
func (r *relay) send(ctx context.Context, method string, body, response any) error {
	if r.uriErr != nil {
		return r.uriErr
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.uri+registrationPath, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	origin, err := r.originGetter()
	if err != nil {
		return fmt.Errorf("get origin: %w", err)
	}
	// The relay calls the server back at the origin plus a path.
	origin = strings.TrimRight(origin, "/")
	if origin == "" {
		return errors.New("missing origin: server URL is not set")
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.c.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, str.MaxErrorResponseBytes+1))
		return &RelayStatusError{Code: resp.StatusCode, RespErr: str.TruncateErrorResponse(string(bodyBytes))}
	}
	if response == nil {
		// The status already says the call succeeded; draining only lets the connection be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxRelayResponseBytes))
		return nil
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRelayResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(respBody) > maxRelayResponseBytes {
		return fmt.Errorf("response body exceeds %d bytes", maxRelayResponseBytes)
	}
	if len(respBody) == 0 {
		return errors.New("decode response: empty response body")
	}
	if err := json.Unmarshal(respBody, response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// RelayStatusError is a non-2xx relay response, with the remote body so callers can
// surface it.
type RelayStatusError struct {
	Code    int
	RespErr string
}

func (e *RelayStatusError) Error() string {
	if e.RespErr == "" {
		return fmt.Sprintf("%d", e.Code)
	}
	return fmt.Sprintf("%d: %s", e.Code, e.RespErr)
}

// StatusCode returns the remote HTTP status code.
func (e *RelayStatusError) StatusCode() int { return e.Code }

// Body returns the remote response body.
func (e *RelayStatusError) Body() string { return e.RespErr }

// relayNotFoundError is a 404 on Deregister, where it means fleetdm.com has no matching
// registration.
type relayNotFoundError struct {
	*RelayStatusError
}

func (e *relayNotFoundError) IsNotFound() bool    { return true }
func (e *relayNotFoundError) IsClientError() bool { return true }
func (e *relayNotFoundError) Unwrap() error       { return e.RelayStatusError }
