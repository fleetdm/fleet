// Package fleetdm_proxy is the HTTP client for the services fleetdm.com hosts for Fleet
// servers, such as the Microsoft and Google conditional access proxies. Requests carry
// the Fleet server URL in the Origin header, which fleetdm.com uses to identify the
// server and call it back.
package fleetdm_proxy

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
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// maxResponseBytes bounds a successful response body.
const maxResponseBytes = 1 << 20

// Client sends JSON requests to a fleetdm.com service.
type Client struct {
	uri          string
	originGetter func() (string, error)
	c            *http.Client
}

// New creates a Client for the service at uri. originGetter returns this Fleet server's
// URL.
func New(uri string, originGetter func() (string, error), opts ...fleethttp.ClientOpt) (*Client, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid uri %q: scheme and host are required", uri)
	}
	return &Client{
		uri:          uri,
		originGetter: originGetter,
		c:            fleethttp.NewClient(opts...),
	}, nil
}

// Get sends a GET to path with the given query (may be nil) and decodes the JSON
// response into response (nil to ignore it).
func (c *Client) Get(ctx context.Context, path string, query url.Values, response any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, response)
}

// Post sends body as JSON to path and decodes the JSON response into response (nil to
// ignore it).
func (c *Client) Post(ctx context.Context, path string, body, response any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, response)
}

// Delete sends a DELETE to path with the given query (may be nil) and decodes the JSON
// response into response (nil to ignore it).
func (c *Client) Delete(ctx context.Context, path string, query url.Values, response any) error {
	return c.do(ctx, http.MethodDelete, path, query, nil, response)
}

// do sends the request and decodes a 2xx JSON response into response. An empty 2xx body
// is only accepted when response is nil. Any other status returns a *StatusError.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, response any) error {
	reqURL := c.uri + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	origin, err := c.originGetter()
	if err != nil {
		return fmt.Errorf("get origin: %w", err)
	}
	if origin == "" {
		return errors.New("missing origin: server URL is not set")
	}
	req.Header.Set("Origin", origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.c.Do(req)
	if err != nil {
		// The query carries the server secret and a transport error would print
		// the full URL.
		return fmt.Errorf("send request: %w", platform_http.MaskURLError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newStatusError(resp)
	}
	// Read the whole body even when the caller ignores it so the connection can
	// be reused.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(respBody) > maxResponseBytes {
		return fmt.Errorf("response body exceeds %d bytes", maxResponseBytes)
	}
	if response == nil {
		return nil
	}
	if len(respBody) == 0 {
		return errors.New("decode response: empty response body")
	}
	if err := json.Unmarshal(respBody, response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// StatusError is a non-2xx response, with the remote body so callers can surface it.
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

// newStatusError reads up to str.MaxErrorResponseBytes of the response body. When the
// body is larger the stored string ends with " [truncated]".
func newStatusError(resp *http.Response) *StatusError {
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, str.MaxErrorResponseBytes+1))
	return &StatusError{Code: resp.StatusCode, RespErr: str.TruncateErrorResponse(string(bodyBytes))}
}

// NotFoundError is a 404 from a call where it means the record doesn't exist.
type NotFoundError struct {
	*StatusError
}

func (e *NotFoundError) IsNotFound() bool    { return true }
func (e *NotFoundError) IsClientError() bool { return true }
func (e *NotFoundError) Unwrap() error       { return e.StatusError }

// NotFoundOn404 turns a 404 StatusError into a NotFoundError, which satisfies
// fleet.IsNotFound. Only calls where a 404 means "no such record" use it, so a 404 from a
// misrouted request elsewhere isn't mistaken for one.
func NotFoundOn404(err error) error {
	if se, ok := errors.AsType[*StatusError](err); ok && se.Code == http.StatusNotFound {
		return &NotFoundError{StatusError: se}
	}
	return err
}
