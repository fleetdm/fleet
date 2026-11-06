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

const maxRelayResponseBytes = 1 << 20

type RelayOption func(*relayOptions)

type relayOptions struct {
	allowHTTP bool
}

// AllowHTTP accepts a plain HTTP relay URI for a dev or test mock. Otherwise HTTPS is
// required, since the relay exchanges the server secret.
func AllowHTTP() RelayOption {
	return func(o *relayOptions) { o.allowHTTP = true }
}

// relay sends this server's URL as the Origin, which fleetdm.com calls back.
type relay struct {
	uri          string
	uriErr       error
	originGetter func() (string, error)
	c            *http.Client
}

// NewRelay fails on an unparsable uri only; an unusable one fails each request instead,
// so an unused integration doesn't stop the server from starting.
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
	// A followed 307 or 308 would resend the server secret to another host.
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
	// TODO: may be unneeded; the Android app is its only consumer.
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

// send decodes a 2xx response into response, which may be nil. Other statuses return a
// *RelayStatusError.
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
	// The relay appends paths to the origin.
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
		// Drain so the connection can be reused.
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

// RelayStatusError is a non-2xx relay response with its body.
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

func (e *RelayStatusError) StatusCode() int { return e.Code }

func (e *RelayStatusError) Body() string { return e.RespErr }

// relayNotFoundError is a 404 on Deregister: no matching registration.
type relayNotFoundError struct {
	*RelayStatusError
}

func (e *relayNotFoundError) IsNotFound() bool    { return true }
func (e *relayNotFoundError) IsClientError() bool { return true }
func (e *relayNotFoundError) Unwrap() error       { return e.RelayStatusError }
