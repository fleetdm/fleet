package msgraph

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Error is a failed Microsoft Graph call, classified by what the caller should do about it.
//
// The classification keys on the HTTP status, not on the error code string, deliberately: Graph answers the same
// underlying "you lack this permission" cause with different codes depending on the endpoint family
// (Authorization_RequestDenied on directory endpoints, Forbidden on Intune endpoints), so matching on code would be
// fragile.
type Error struct {
	StatusCode int
	// Code and Message are Graph's own error fields, kept for display and logging.
	Code    string
	Message string
	// RetryAfter is populated from the Retry-After header when Graph throttles.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("microsoft graph: %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("microsoft graph: %d: %s", e.StatusCode, e.Message)
}

// IsAuthError reports whether the credential itself was rejected. The admin has to supply a new secret.
func (e *Error) IsAuthError() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsPermissionError reports whether the token was accepted but the app lacks the required permission or admin consent.
// The admin has to grant DeviceManagementServiceConfig.Read.All and consent to it.
func (e *Error) IsPermissionError() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsTransient reports whether the call should simply be retried. Throttling and server errors must never be treated as
// a bad credential, or a Microsoft outage would raise a credential alarm on every Fleet deployment at once.
func (e *Error) IsTransient() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= http.StatusInternalServerError
}

// graphErrorBody is Graph's standard error envelope.
type graphErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newGraphError(resp *http.Response, body []byte) *Error {
	graphErr := &Error{StatusCode: resp.StatusCode, Message: string(body)}

	var parsed graphErrorBody
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Code != "" {
		graphErr.Code = parsed.Error.Code
		graphErr.Message = parsed.Error.Message
	}

	// Retry-After is seconds on this API. A malformed or absent header just leaves the caller to pick its own backoff.
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
			graphErr.RetryAfter = time.Duration(secs) * time.Second
		}
	}

	return graphErr
}

// AsError extracts a *Error from a wrapped error chain, so callers can classify a failure without knowing how deeply
// it was wrapped.
func AsError(err error) (*Error, bool) {
	return errors.AsType[*Error](err)
}
