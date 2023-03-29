package fleet

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

var (
	ErrNoContext             = errors.New("context key not set")
	ErrPasswordResetRequired = &passwordResetRequiredError{}
	ErrMissingLicense        = &licenseError{}
	ErrMDMNotConfigured      = &MDMNotConfiguredError{}
)

// ErrWithInternal is an interface for errors that include extra "internal"
// information that should be logged in server logs but not sent to clients.
type ErrWithInternal interface {
	error
	// Internal returns the error string that must only be logged internally,
	// not returned to the client.
	Internal() string
}

// ErrWithInternal is an interface for errors that include additional logging
// fields that should be logged in server logs but not sent to clients.
type ErrWithLogFields interface {
	error
	// LogFields returns the additional log fields to add, which should come in
	// key, value pairs (as used in go-kit log).
	LogFields() []interface{}
}

// ErrWithRetryAfter is an interface for errors that should set a specific HTTP
// Header Retry-After value (see
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)
type ErrWithRetryAfter interface {
	error
	// RetryAfter returns the number of seconds to wait before retry.
	RetryAfter() int
}

type invalidArgWithStatusError struct {
	InvalidArgumentError
	code int
}

func (e invalidArgWithStatusError) Status() int {
	if e.code == 0 {
		// 422 is the default code for invalid args
		return http.StatusUnprocessableEntity
	}
	return e.code
}

// ErrorUUIDer is the interface for errors that contain a UUID.
type ErrorUUIDer interface {
	// UUID returns the error's UUID.
	UUID() string
}

// ErrorWithUUID can be embedded to error types to implement ErrorUUIDer.
type ErrorWithUUID struct {
	uuid string
}

var _ ErrorUUIDer = (*ErrorWithUUID)(nil)

// UUID implements the ErrorUUIDer interface.
func (e *ErrorWithUUID) UUID() string {
	if e.uuid == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			panic(err)
		}
		e.uuid = uuid.String()
	}
	return e.uuid
}

// InvalidArgumentError is the error returned when invalid data is presented to
// a service method.
type InvalidArgumentError struct {
	Errors []InvalidArgument

	ErrorWithUUID
}

// InvalidArgument is the details about a single invalid argument.
type InvalidArgument struct {
	name   string
	reason string
}

// NewInvalidArgumentError returns a InvalidArgumentError with at least
// one error.
func NewInvalidArgumentError(name, reason string) *InvalidArgumentError {
	var invalid InvalidArgumentError
	invalid.Append(name, reason)
	return &invalid
}

func (e *InvalidArgumentError) Append(name, reason string) {
	e.Errors = append(e.Errors, InvalidArgument{
		name:   name,
		reason: reason,
	})
}

func (e *InvalidArgumentError) Appendf(name, reasonFmt string, args ...interface{}) {
	e.Append(name, fmt.Sprintf(reasonFmt, args...))
}

// WithStatus returns an error that combines the InvalidArgumentError
// with a custom status code.
func (e InvalidArgumentError) WithStatus(code int) error {
	return invalidArgWithStatusError{e, code}
}

func (e *InvalidArgumentError) HasErrors() bool {
	return len(e.Errors) != 0
}

// Error implements the error interface.
func (e InvalidArgumentError) Error() string {
	switch len(e.Errors) {
	case 0:
		return "validation failed"
	case 1:
		return fmt.Sprintf("validation failed: %s %s", e.Errors[0].name, e.Errors[0].reason)
	default:
		return fmt.Sprintf("validation failed: %s %s and %d other errors", e.Errors[0].name, e.Errors[0].reason,
			len(e.Errors))
	}
}

func (e InvalidArgumentError) Invalid() []map[string]string {
	var invalid []map[string]string
	for _, i := range e.Errors {
		invalid = append(invalid, map[string]string{"name": i.name, "reason": i.reason})
	}
	return invalid
}

// BadRequestError is an error type that generates a 400 status code.
type BadRequestError struct {
	Message     string
	InternalErr error

	ErrorWithUUID
}

// Error returns the error message.
func (e *BadRequestError) Error() string {
	return e.Message
}

// This implements the interface required by the server/service package logic
// to determine the status code to return to the client.
func (e *BadRequestError) BadRequestError() []map[string]string {
	return nil
}

func (e BadRequestError) Internal() string {
	if e.InternalErr == nil {
		return ""
	}
	return e.InternalErr.Error()
}

type AuthFailedError struct {
	// internal is the reason that should only be logged internally
	internal string

	ErrorWithUUID
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

	ErrorWithUUID
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

type AuthHeaderRequiredError struct {
	// internal is the reason that should only be logged internally
	internal string

	ErrorWithUUID
}

func NewAuthHeaderRequiredError(internal string) *AuthHeaderRequiredError {
	return &AuthHeaderRequiredError{
		internal: internal,
	}
}

func (e AuthHeaderRequiredError) Error() string {
	return "Authorization header required"
}

func (e AuthHeaderRequiredError) Internal() string {
	return e.internal
}

func (e AuthHeaderRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// PermissionError, set when user is authenticated, but not allowed to perform action
type PermissionError struct {
	message string

	ErrorWithUUID
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

// licenseError is returned when the application is not properly licensed.
type licenseError struct {
	ErrorWithUUID
}

func (e licenseError) Error() string {
	return "Requires Fleet Premium license"
}

func (e licenseError) StatusCode() int {
	return http.StatusPaymentRequired
}

type passwordResetRequiredError struct {
	ErrorWithUUID
}

func (e passwordResetRequiredError) Error() string {
	return "password reset required"
}

func (e passwordResetRequiredError) StatusCode() int {
	return http.StatusUnauthorized
}

// MDMNotConfiguredError is used when an MDM endpoint or resource is accessed
// without having MDM correctly configured.
type MDMNotConfiguredError struct{}

// Status implements the kithttp.StatusCoder interface so we can customize the
// HTTP status code of the response returning this error.
func (e *MDMNotConfiguredError) StatusCode() int {
	return http.StatusBadRequest
}

func (e *MDMNotConfiguredError) Error() string {
	return "MDM features aren't turned on in Fleet. For more information about setting up MDM, please visit https://fleetdm.com/docs/using-fleet/mobile-device-management"
}

// Error is a user facing error (API user). It's meant to be used for errors that are
// related to fleet logic specifically. Other errors, such as mysql errors, shouldn't
// be translated to this.
type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`

	ErrorWithUUID
}

const (
	// ErrNoRoleNeeded is the error number for valid role needed
	ErrNoRoleNeeded = 1
	// ErrNoOneAdminNeeded is the error number when all admins are about to be removed
	ErrNoOneAdminNeeded = 2
	// ErrNoUnknownTranslate is returned when an item type in the translate payload is unknown
	ErrNoUnknownTranslate = 3
	// ErrAPIOnlyRole is returned when a selected role for a user is for API only users.
	ErrAPIOnlyRole = 4
	// ErrTeamGitOpsRoleMustBeUnique is returned when attempting to set GitOps role mixed with other
	// roles in other teams. E.g. A user cannot be GitOps on team A, and maintainer on team B.
	ErrTeamGitOpsRoleMustBeUnique = 4
)

// NewError returns a fleet error with the code and message specified
func NewError(code int, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorf returns a fleet error with the code, and message formatted
// based on the format string and args specified
func NewErrorf(code int, format string, args ...interface{}) error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func (ge *Error) Error() string {
	return ge.Message
}

// UserMessageError is an error that adds the UserMessage interface
// implementation.
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

var rxJSONUnknownField = regexp.MustCompile(`^json: unknown field "(.+)"$`)

// IsJSONUnknownFieldError returns true if err is a JSON unknown field error.
// There is no exported type or value for this error, so we have to match the
// error message.
func IsJSONUnknownFieldError(err error) bool {
	return rxJSONUnknownField.MatchString(err.Error())
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

// StatusCode implements the kithttp.StatusCoder interface to return the status
// code to use in HTTP API responses.
func (e UserMessageError) StatusCode() int {
	if e.statusCode > 0 {
		return e.statusCode
	}
	return http.StatusUnprocessableEntity
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
