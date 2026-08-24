package msgraph

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/oauth2"
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
	// Err is the underlying cause, when there is one.
	Err error
}

// Unwrap exposes the underlying cause.
func (e *Error) Unwrap() error { return e.Err }

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

// UserFacingMessage turns a Graph failure into a single sentence an admin can act on, discarding the internal wrap
// chain that an error accumulates on its way up.
func UserFacingMessage(err error) string {
	graphErr, ok := AsError(err)
	if !ok {
		return ""
	}
	switch {
	case graphErr.IsPermissionError():
		return "Microsoft Graph denied the request. Grant the app registration the DeviceManagementServiceConfig.Read.All " +
			"application permission and grant admin consent for your tenant."
	case graphErr.IsAuthError():
		return "Microsoft Graph rejected the credential. Check the tenant ID, client ID, and client secret."
	case graphErr.IsTransient():
		return fmt.Sprintf("Microsoft Graph is temporarily unavailable (%d).", graphErr.StatusCode)
	default:
		return fmt.Sprintf("Microsoft Graph returned an error (%d): %s", graphErr.StatusCode, graphErr.Message)
	}
}

// AsError extracts the Graph error from a possibly-wrapped error. Callers classify a failure by unwrapping it here
// rather than type-asserting themselves, so the unwrap and the classification stay in one place.
func AsError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	return errors.AsType[*Error](err)
}

// CredentialRejected reports whether a failure means the credential itself needs an admin's attention, as opposed to
// Microsoft being temporarily unavailable. A non-Graph failure is never a rejection: a DNS or dial error says nothing
// about the credential.
func CredentialRejected(err error) bool {
	graphErr, ok := AsError(err)
	return ok && (graphErr.IsAuthError() || graphErr.IsPermissionError())
}

// graphErrorBody is Graph's standard error envelope.
type graphErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// newTokenError converts an OAuth2 token-endpoint failure into an *Error so callers classify it exactly as they classify
// a Graph response. The most common misconfiguration is a wrong or expired client secret, fails at Entra's token
// endpoint before Graph is ever reached.
func newTokenError(retrieveErr *oauth2.RetrieveError) *Error {
	// RFC 6749 specifies 401 for invalid_client, but providers are inconsistent and some answer 400. Pin it so the
	// credential is classified as rejected either way.
	status := http.StatusUnauthorized
	if retrieveErr.ErrorCode != "invalid_client" && retrieveErr.Response != nil {
		status = retrieveErr.Response.StatusCode
	}

	// Bounded for the same reason a Graph body is.
	message := retrieveErr.ErrorDescription
	if message == "" {
		message = truncateBody(retrieveErr.Body)
	}

	return &Error{StatusCode: status, Code: retrieveErr.ErrorCode, Message: message, Err: retrieveErr}
}

// maxErrorBodyBytes bounds how much of an unparseable response body is kept in the message. A 5xx from an edge proxy
// can return a large HTML page, and this string ends up in logs and in the sync error surfaced to the admin.
const maxErrorBodyBytes = 512

func newGraphError(resp *http.Response, body []byte) *Error {
	graphErr := &Error{StatusCode: resp.StatusCode, Message: truncateBody(body)}

	var parsed graphErrorBody
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Code != "" {
		graphErr.Code = parsed.Error.Code
		graphErr.Message = parsed.Error.Message
	}

	graphErr.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))

	return graphErr
}

func truncateBody(body []byte) string {
	if len(body) > maxErrorBodyBytes {
		return string(body[:maxErrorBodyBytes]) + "... (truncated)"
	}
	return string(body)
}

// parseRetryAfter handles both forms RFC 7231 allows for the header. Graph sends delta-seconds in practice, but an
// HTTP-date would otherwise be read as zero backoff, which is worse than no value at all.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		if d := time.Until(when); d > 0 {
			return d
		}
	}
	return 0
}
