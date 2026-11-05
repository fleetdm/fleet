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
	if _, err := url.Parse(uri); err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}
	return &Client{
		uri:          uri,
		originGetter: originGetter,
		c:            fleethttp.NewClient(opts...),
	}, nil
}

// Do sends a request to path with the given query (may be nil) and JSON body (nil for
// none), and decodes a 2xx JSON response into response (nil to ignore it). Any other
// status returns a *StatusError.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, response any) error {
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
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newStatusError(resp)
	}
	if response == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(response); err != nil {
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
