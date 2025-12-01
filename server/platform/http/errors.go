package http

import (
	"errors"
	"net/http"

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
	LogFields() []interface{}
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
func (e *BadRequestError) Internal() string {
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
func (e *UserMessageError) StatusCode() int {
	if e.statusCode > 0 {
		return e.statusCode
	}
	return http.StatusUnprocessableEntity
}

// UserMessage returns the user-friendly error message.
func (e *UserMessageError) UserMessage() string {
	return e.Error()
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

// OrderDirection defines the order direction for list queries.
type OrderDirection int

const (
	OrderAscending OrderDirection = iota
	OrderDescending
)
