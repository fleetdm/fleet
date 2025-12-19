package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ErrWithInternal defines an interface for errors that have an internal message
// that should only be logged, not returned to the client.
type ErrWithInternal interface {
	error
	// Internal returns the error string that must only be logged internally,
	// not returned to the client.
	Internal() string
}

// ErrWithLogFields defines an interface for errors that have additional log fields.
type ErrWithLogFields interface {
	error
	// LogFields returns the additional log fields to add, which should come in
	// key, value pairs (as used in go-kit log).
	LogFields() []any
}

// ErrorUUIDer defines an interface for errors that have a UUID for tracking.
type ErrorUUIDer interface {
	// UUID returns the error's UUID.
	UUID() string
}

// ErrorWithUUID can be embedded in error types to implement ErrorUUIDer.
type ErrorWithUUID struct {
	uuid string
}

var _ ErrorUUIDer = (*ErrorWithUUID)(nil)

// UUID implements the ErrorUUIDer interface.
func (e *ErrorWithUUID) UUID() string {
	if e.uuid == "" {
		u, err := uuid.NewRandom()
		if err != nil {
			panic(err)
		}
		e.uuid = u.String()
	}
	return e.uuid
}

// BadRequestError is the error returned when the request is invalid.
type BadRequestError struct {
	Message     string
	InternalErr error

	ErrorWithUUID
}

// Error returns the error message.
func (e *BadRequestError) Error() string {
	return e.Message
}

// BadRequestError implements the interface required by the server/service package logic
// to determine the status code to return to the client.
func (e *BadRequestError) BadRequestError() []map[string]string {
	return nil
}

// Internal implements the ErrWithInternal interface.
func (e BadRequestError) Internal() string {
	if e.InternalErr != nil {
		return e.InternalErr.Error()
	}
	return ""
}

// UserMessageError is an error that wraps another error with a user-friendly message.
type UserMessageError struct {
	error
	statusCode int

	ErrorWithUUID
}

// NewUserMessageError creates a UserMessageError that will translate the
// error message of err to a user-friendly form. If statusCode is > 0, it
// will be used as the HTTP status code for the error, otherwise it defaults
// to http.StatusUnprocessableEntity (422).
func NewUserMessageError(err error, statusCode int) *UserMessageError {
	if err == nil {
		return nil
	}
	return &UserMessageError{
		error:      err,
		statusCode: statusCode,
	}
}

// StatusCode returns the HTTP status code for this error.
func (e UserMessageError) StatusCode() int {
	if e.statusCode > 0 {
		return e.statusCode
	}
	return http.StatusUnprocessableEntity
}

var rxJSONUnknownField = regexp.MustCompile(`^json: unknown field "(.+)"$`)

// IsJSONUnknownFieldError returns true if err is a JSON unknown field error.
// There is no exported type or value for this error, so we have to match the
// error message.
func IsJSONUnknownFieldError(err error) bool {
	return rxJSONUnknownField.MatchString(err.Error())
}

// GetJSONUnknownField returns the unknown field name from a JSON unknown field error.
func GetJSONUnknownField(err error) *string {
	errCause := Cause(err)
	if IsJSONUnknownFieldError(errCause) {
		substr := rxJSONUnknownField.FindStringSubmatch(errCause.Error())
		return &substr[1]
	}
	return nil
}

// UserMessage implements the user-friendly translation of the error if its
// root cause is one of the supported types, otherwise it returns the error
// message.
func (e UserMessageError) UserMessage() string {
	cause := Cause(e.error)
	switch cause := cause.(type) {
	case *json.UnmarshalTypeError:
		var sb strings.Builder
		curType := cause.Type
		for curType.Kind() == reflect.Slice || curType.Kind() == reflect.Array {
			sb.WriteString("array of ")
			curType = curType.Elem()
		}
		sb.WriteString(curType.Name())
		if curType != cause.Type {
			// it was an array
			sb.WriteString("s")
		}

		return fmt.Sprintf("invalid value type at '%s': expected %s but got %s", cause.Field, sb.String(), cause.Value)

	default:
		// there's no specific error type for the strict json mode
		// (DisallowUnknownFields), so resort to message-matching.
		if matches := rxJSONUnknownField.FindStringSubmatch(cause.Error()); matches != nil {
			return fmt.Sprintf("unsupported key provided: %q", matches[1])
		}
		return e.Error()
	}
}

// Cause returns the root error in err's chain.
func Cause(err error) error {
	for {
		uerr := errors.Unwrap(err)
		if uerr == nil {
			return err
		}
		err = uerr
	}
}

// ErrWithRetryAfter is an interface for errors that should set a specific HTTP
// Header Retry-After value (see
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)
type ErrWithRetryAfter interface {
	error
	// RetryAfter returns the number of seconds to wait before retry.
	RetryAfter() int
}

// ForeignKeyError is an interface for errors caused by foreign key constraint violations.
type ForeignKeyError interface {
	error
	IsForeignKey() bool
}

// IsForeignKey returns true if err is a foreign key constraint violation.
func IsForeignKey(err error) bool {
	var fke ForeignKeyError
	if errors.As(err, &fke) {
		return fke.IsForeignKey()
	}
	return false
}

// Error is a generic error type with a code and message.
type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`

	ErrorWithUUID
}

// Error returns the error message.
func (e *Error) Error() string {
	return e.Message
}

// ErrWithIsClientError is an interface for errors that explicitly specify
// whether they are client errors or not. By default, errors are treated as
// server errors.
type ErrWithIsClientError interface {
	error
	IsClientError() bool
}

// AuthFailedError is returned when authentication fails.
type AuthFailedError struct {
	// internal is the reason that should only be logged internally
	internal string

	ErrorWithUUID
}

// NewAuthFailedError creates a new AuthFailedError.
func NewAuthFailedError(internal string) *AuthFailedError {
	return &AuthFailedError{internal: internal}
}

// Error implements the error interface.
func (e AuthFailedError) Error() string {
	return "Authentication failed"
}

// Internal implements ErrWithInternal.
func (e AuthFailedError) Internal() string {
	return e.internal
}

// StatusCode implements kithttp.StatusCoder.
func (e AuthFailedError) StatusCode() int {
	return http.StatusUnauthorized
}

// AuthRequiredError is returned when authentication is required.
type AuthRequiredError struct {
	// internal is the reason that should only be logged internally
	internal string

	ErrorWithUUID
}

// NewAuthRequiredError creates a new AuthRequiredError.
func NewAuthRequiredError(internal string) *AuthRequiredError {
	return &AuthRequiredError{internal: internal}
}

// Error implements the error interface.
func (e AuthRequiredError) Error() string {
	return "Authentication required"
}

// Internal implements ErrWithInternal.
func (e AuthRequiredError) Internal() string {
	return e.internal
}

// StatusCode implements kithttp.StatusCoder.
func (e AuthRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// AuthHeaderRequiredError is returned when an authorization header is required.
type AuthHeaderRequiredError struct {
	// internal is the reason that should only be logged internally
	internal string

	ErrorWithUUID
}

// NewAuthHeaderRequiredError creates a new AuthHeaderRequiredError.
func NewAuthHeaderRequiredError(internal string) *AuthHeaderRequiredError {
	return &AuthHeaderRequiredError{
		internal: internal,
	}
}

// Error implements the error interface.
func (e AuthHeaderRequiredError) Error() string {
	return "Authorization header required"
}

// Internal implements ErrWithInternal.
func (e AuthHeaderRequiredError) Internal() string {
	return e.internal
}

// StatusCode implements kithttp.StatusCoder.
func (e AuthHeaderRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// ErrPasswordResetRequired is returned when a password reset is required.
var ErrPasswordResetRequired = &passwordResetRequiredError{}

type passwordResetRequiredError struct {
	ErrorWithUUID
}

// Error implements the error interface.
func (e passwordResetRequiredError) Error() string {
	return "password reset required"
}

// StatusCode implements kithttp.StatusCoder.
func (e passwordResetRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// OrderDirection defines the order direction for list queries.
type OrderDirection int

const (
	OrderAscending OrderDirection = iota
	OrderDescending
)

// ForbiddenErrorMessage is the error message that should be returned to
// clients when an action is forbidden. It is intentionally vague to prevent
// disclosing information that a client should not have access to.
const ForbiddenErrorMessage = "forbidden"

// CheckMissing is the error to return when no authorization check was performed
// by the service.
type CheckMissing struct {
	response any

	ErrorWithUUID
}

// CheckMissingWithResponse creates a new error indicating the authorization
// check was missed, and including the response for further analysis by the error
// encoder.
func CheckMissingWithResponse(response any) *CheckMissing {
	return &CheckMissing{response: response}
}

// Error implements the error interface.
func (e *CheckMissing) Error() string {
	return ForbiddenErrorMessage
}

// Internal implements the ErrWithInternal interface.
func (e *CheckMissing) Internal() string {
	return "Missing authorization check"
}

// Response returns the response that was generated before the authorization
// check was found to be missing.
func (e *CheckMissing) Response() any {
	return e.response
}
