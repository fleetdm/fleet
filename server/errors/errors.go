package errors

import (
	goerrs "errors"
	"net/http"
)

var (
	// ErrNotFound is returned when the datastore resource cannot be found
	ErrNotFound = goerrs.New("resource not found")

	// ErrExists is returned when creating a datastore resource that already exists
	ErrExists = goerrs.New("resource already created")
)

// Kolide's internal representation for errors. It can be used to wrap another
// error (stored in Err), and additionally contains fields for public
// (PublicMessage) and private (PrivateMessage) error messages as well as the
// HTTP status code (StatusCode) corresponding to the error. Extra holds extra
// information that will be inserted as top level key/value pairs in the error
// response.
type KolideError struct {
	Err            error
	StatusCode     int
	PublicMessage  string
	PrivateMessage string
	Extra          map[string]interface{}
}

// Implementation of error interface
func (e *KolideError) Error() string {
	return e.PublicMessage
}

// Create a new KolideError specifying the public and private messages. The
// status code will be set to 500.
func New(publicMessage, privateMessage string) *KolideError {
	return &KolideError{
		StatusCode:     http.StatusInternalServerError,
		PublicMessage:  publicMessage,
		PrivateMessage: privateMessage,
	}
}

// Create a new KolideError specifying the HTTP status, and public and private
// messages.
func NewWithStatus(status int, publicMessage, privateMessage string) *KolideError {
	return &KolideError{
		StatusCode:     status,
		PublicMessage:  publicMessage,
		PrivateMessage: privateMessage,
	}
}

// Create a new KolideError from an error type. The public message and status
// code should be specified, while the private message will be drawn from
// err.Error()
func NewFromError(err error, status int, publicMessage string) *KolideError {
	return &KolideError{
		Err:            err,
		StatusCode:     status,
		PublicMessage:  publicMessage,
		PrivateMessage: err.Error(),
	}
}

// Wrap a DB error with the extra KolideError decorations
func DatabaseError(err error) *KolideError {
	return NewFromError(err, http.StatusInternalServerError, "Database error")
}

// Wrap a server error with the extra KolideError decorations
func InternalServerError(err error) *KolideError {
	return NewFromError(err, http.StatusInternalServerError, "Internal server error")
}

// The status code returned for validation errors. Inspired by the Github API.
const StatusUnprocessableEntity = 422
