package service

import (
	"fmt"
	"net/http"
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

type invalidArgumentError []invalidArgument
type invalidArgument struct {
	name   string
	reason string
}

// newInvalidArgumentError returns a invalidArgumentError with at least
// one error.
func newInvalidArgumentError(name, reason string) *invalidArgumentError {
	var invalid invalidArgumentError
	invalid = append(invalid, invalidArgument{
		name:   name,
		reason: reason,
	})
	return &invalid
}

func (e *invalidArgumentError) Append(name, reason string) {
	*e = append(*e, invalidArgument{
		name:   name,
		reason: reason,
	})
}
func (e *invalidArgumentError) Appendf(name, reasonFmt string, args ...interface{}) {
	*e = append(*e, invalidArgument{
		name:   name,
		reason: fmt.Sprintf(reasonFmt, args...),
	})
}

func (e *invalidArgumentError) HasErrors() bool {
	return len(*e) != 0
}

// invalidArgumentError is returned when one or more arguments are invalid.
func (e invalidArgumentError) Error() string {
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

func (e invalidArgumentError) Invalid() []map[string]string {
	var invalid []map[string]string
	for _, i := range e {
		invalid = append(invalid, map[string]string{"name": i.name, "reason": i.reason})
	}
	return invalid
}

type authFailedError struct {
	// internal is the reason that should only be logged internally
	internal string
}

func (e authFailedError) Error() string {
	return "Authentication failed"
}

func (e authFailedError) Internal() string {
	return e.internal
}

func (e authFailedError) StatusCode() int {
	return http.StatusUnauthorized
}

type authRequiredError struct {
	// internal is the reason that should only be logged internally
	internal string
}

func (e authRequiredError) Error() string {
	return "Authentication required"
}

func (e authRequiredError) Internal() string {
	return e.internal
}

func (e authRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// permissionError, set when user is authenticated, but not allowed to perform action
type permissionError struct {
	message string
	badArgs []invalidArgument
}

func newPermissionError(name, reason string) permissionError {
	return permissionError{
		badArgs: []invalidArgument{
			invalidArgument{
				name:   name,
				reason: reason,
			},
		},
	}
}

func (e permissionError) Error() string {
	switch len(e.badArgs) {
	case 0:
	case 1:
		e.message = fmt.Sprintf("unauthorized: %s",
			e.badArgs[0].reason,
		)
	default:
		e.message = fmt.Sprintf("unauthorized: %s and %d other errors",
			e.badArgs[0].reason,
			len(e.badArgs),
		)
	}
	if e.message == "" {
		return "unauthorized"
	}
	return e.message
}

func (e permissionError) PermissionError() []map[string]string {
	var forbidden []map[string]string
	if len(e.badArgs) == 0 {
		forbidden = append(forbidden, map[string]string{"reason": e.Error()})
		return forbidden
	}
	for _, arg := range e.badArgs {
		forbidden = append(forbidden, map[string]string{
			"name":   arg.name,
			"reason": arg.reason,
		})
	}
	return forbidden

}
