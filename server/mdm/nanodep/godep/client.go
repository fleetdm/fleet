// Package godep provides Go methods and structures for talking to individual DEP API endpoints.
package godep

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
)

const (
	mediaType = "application/json;charset=UTF8"
	// Apple blocks the default nanodep user agent with newer protocol versions
	// cf https://github.com/micromdm/nanodep/issues/42
	userAgent = "fleetdm/nanodep"
)

// HTTPError encapsulates an HTTP response error from the DEP requests.
// The API returns error information in the request body.
type HTTPError struct {
	Body       []byte
	Status     string
	StatusCode int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("DEP HTTP error: %s: %s", e.Status, string(e.Body))
}

// NewHTTPError creates and returns a new HTTPError from r. Note this reads
// all of r.Body and the caller is responsible for closing it.
func NewHTTPError(r *http.Response) error {
	body, readErr := io.ReadAll(r.Body)
	err := &HTTPError{
		Body:       body,
		Status:     r.Status,
		StatusCode: r.StatusCode,
	}
	if readErr != nil {
		return fmt.Errorf("reading body of DEP HTTP error: %v: %w", err, readErr)
	}
	return err
}

// httpErrorContains checks if err is an HTTPError and contains body and a
// matching status code. With the depsim DEP simulator the body strings are
// returned with surrounding quotes. i.e. `"INVALID_CURSOR"` vs. just
// `INVALID_CURSOR` so we search the body data for the string vs. matching.
func httpErrorContains(err error, status int, s string) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == status && bytes.Contains(httpErr.Body, []byte(s)) {
		return true
	}
	return false
}

// authErrorContains is the same as httpErrorContains except that it checks if
// err is an depclient.AuthError instead of HTTPError.
func authErrorContains(err error, status int, s string) bool {
	var authErr *depclient.AuthError
	if errors.As(err, &authErr) && authErr.StatusCode == status && bytes.Contains(authErr.Body, []byte(s)) {
		return true
	}
	return false
}

// ClientStorage provides the required data needed to connect to the Apple DEP APIs.
type ClientStorage interface {
	depclient.AuthTokensRetriever
	depclient.ConfigRetriever
}

// Client represents an Apple DEP API client identified by a single DEP name.
type Client struct {
	store ClientStorage

	// an HTTP client that handles DEP API authentication and session management
	client    depclient.Doer
	afterHook func(ctx context.Context, err error) error

	// Optional logger for debugging
	logger log.Logger
}

// ClientOption defines the functional options type for NewClient.
type ClientOption func(*Client)

// WithAfterHook installs a hook function that is called with the error
// resulting from any request, after transformation of the response's body to
// an HTTPError if needed. It gets called regardless of success or failure of
// the request, with a nil error if it succeeded. It can return a new error to
// be returned by the Client, or the original error.
func WithAfterHook(hook func(ctx context.Context, err error) error) ClientOption {
	return func(c *Client) {
		c.afterHook = hook
	}
}

func WithLogger(logger log.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates new Client and reads authentication and config data
// from store. The provided client is copied and modified by wrapping its
// transport in a new NanoDEP transport (which transparently handles
// authentication and session management). If client is nil then
// http.DefaultClient is used.
func NewClient(store ClientStorage, client *http.Client, opts ...ClientOption) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	t := depclient.NewTransport(client.Transport, client, store, nil)
	client = depclient.NewClient(client, t)
	depClient := &Client{
		store:  store,
		client: client,
	}

	for _, opt := range opts {
		opt(depClient)
	}
	return depClient
}

func (c *Client) doWithAfterHook(ctx context.Context, name, method, path string, in interface{}, out interface{}) error {
	req, err := c.do(ctx, name, method, path, in, out)
	if c.afterHook != nil {
		// ensure the afterHook is always called with the same context as the one
		// used for the request (the DEP client will add the name argument to that
		// context, which we care about in the after hook).
		if req != nil {
			ctx = req.Context()
		}
		err = c.afterHook(ctx, err)
	}
	return err
}

// do executes the HTTP request using the client's HTTP client which
// should be using the NanoDEP transport (which handles authentication).
// This frees us to only be concerned about the actual DEP API request.
// We encode in to JSON and decode any returned body as JSON to out.
func (c *Client) do(ctx context.Context, name, method, path string, in interface{}, out interface{}) (*http.Request, error) {
	var body io.Reader
	bodyStr := "[empty]"
	if in != nil {
		bodyBytes, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(bodyBytes)
		bodyStr = string(bodyBytes)
	}

	req, err := depclient.NewRequestWithContext(ctx, name, c.store, method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", mediaType)
	}
	if out != nil {
		req.Header.Set("Accept", mediaType)
	}

	if c.logger != nil {
		c.logger.Debug("msg", "sending request to Apple DEP", "method", method, "url", req.URL.String(), "body", bodyStr)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("msg", "error sending request to Apple DEP", "url", req.URL.String(), "error", err)
		}
		return req, err
	}
	defer resp.Body.Close()

	appleRequestUUID := ""
	if hdr, ok := resp.Header[http.CanonicalHeaderKey("X-Apple-Request-UUID")]; ok {
		for _, hdrValue := range hdr {
			if appleRequestUUID != "" {
				appleRequestUUID += ", "
			}
			appleRequestUUID += hdrValue
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return req, fmt.Errorf("unhandled auth error on request %s: %w", appleRequestUUID, depclient.NewAuthError(resp))
	} else if resp.StatusCode != http.StatusOK {
		return req, NewHTTPError(resp)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return req, err
	}

	limit50KiB := 50 * 1024
	if c.logger != nil {
		responseBodyString := ""
		// This should cover large DEP requests without overwhelming the logs and get the data needed
		// for the most important ones like assign profile responses
		if len(bodyBytes) > limit50KiB {
			responseBodyString = string(bodyBytes[:limit50KiB]) + "...[truncated]"
		} else {
			responseBodyString = string(bodyBytes)
		}
		c.logger.Debug("msg", "Apple DEP Request returned 200 status", "url", req.URL.String(), "apple_request_uuid", appleRequestUUID, "body", responseBodyString)
	}

	if out != nil {
		err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(out)
		if err != nil {
			return req, err
		}
	}

	return req, nil
}
