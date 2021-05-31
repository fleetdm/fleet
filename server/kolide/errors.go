package kolide

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNoContext      = errors.New("context key not set")
	ErrMissingLicense = &LicenseError{}
)

// ErrWithInternal is an interface for errors that include extra "internal"
// information that should be logged in server logs but not sent to clients.
type ErrWithInternal interface {
	error
	// Internal returns the error string that must only be logged internally,
	// not returned to the client.
	Internal() string
}

// ErrWithStatusCode is an interface for errors that should set a specific HTTP
// status when encoding.
type ErrWithStatusCode interface {
	error
	// StatusCode returns the HTTP status code that should be returned.
	StatusCode() int
}

// ErrWithRetryAfter is an interface for errors that should set a specific HTTP
// Header Retry-After value (see
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)
type ErrWithRetryAfter interface {
	error
	// RetryAfter returns the number of seconds to wait before retry.
	RetryAfter() int
}

// InvalidArgumentError is the error returned when invalid data is presented to
// a service method.
type InvalidArgumentError []InvalidArgument

// InvalidArgument is the details about a single invalid argument.
type InvalidArgument struct {
	name   string
	reason string
}

// NewInvalidArgumentError returns a InvalidArgumentError with at least
// one error.
func NewInvalidArgumentError(name, reason string) *InvalidArgumentError {
	var invalid InvalidArgumentError
	invalid = append(invalid, InvalidArgument{
		name:   name,
		reason: reason,
	})
	return &invalid
}

func (e *InvalidArgumentError) Append(name, reason string) {
	*e = append(*e, InvalidArgument{
		name:   name,
		reason: reason,
	})
}
func (e *InvalidArgumentError) Appendf(name, reasonFmt string, args ...interface{}) {
	*e = append(*e, InvalidArgument{
		name:   name,
		reason: fmt.Sprintf(reasonFmt, args...),
	})
}

func (e *InvalidArgumentError) HasErrors() bool {
	return len(*e) != 0
}

// Error implements the error interface.
func (e InvalidArgumentError) Error() string {
	switch len(e) {
	case 0:
		return "validation failed"
	case 1:
		return fmt.Sprintf("validation failed: %s %s", e[0].name, e[0].reason)
	default:
		return fmt.Sprintf("validation failed: %s %s and %d other errors", e[0].name, e[0].reason,
			len(e))
	}
}

func (e InvalidArgumentError) Invalid() []map[string]string {
	var invalid []map[string]string
	for _, i := range e {
		invalid = append(invalid, map[string]string{"name": i.name, "reason": i.reason})
	}
	return invalid
}

type AuthFailedError struct {
	// internal is the reason that should only be logged internally
	internal string
}

func NewAuthFailedError(internal string) *AuthFailedError {
	return &AuthFailedError{internal: internal}
}

func (e AuthFailedError) Error() string {
	return "Authentication failed"
}

func (e AuthFailedError) Internal() string {
	return e.internal
}

func (e AuthFailedError) StatusCode() int {
	return http.StatusUnauthorized
}

type AuthRequiredError struct {
	// internal is the reason that should only be logged internally
	internal string
}

func NewAuthRequiredError(internal string) *AuthRequiredError {
	return &AuthRequiredError{internal: internal}
}

func (e AuthRequiredError) Error() string {
	return "Authentication required"
}

func (e AuthRequiredError) Internal() string {
	return e.internal
}

func (e AuthRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// PermissionError, set when user is authenticated, but not allowed to perform action
type PermissionError struct {
	message string
}

func NewPermissionError(message string) *PermissionError {
	return &PermissionError{message: message}
}

func (e PermissionError) Error() string {
	return e.message
}

func (e PermissionError) PermissionError() []map[string]string {
	var forbidden []map[string]string
	return forbidden
}

// LicenseError is returned when the application is not properly licensed.
type LicenseError struct{}

func (e LicenseError) Error() string {
	return "requires Fleet Basic license"
}

func (e LicenseError) StatusCode() int {
	return http.StatusPaymentRequired
}
