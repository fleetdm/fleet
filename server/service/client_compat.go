package service

import (
	"io"

	"github.com/fleetdm/fleet/v4/client"
)

// Type aliases for base client types (backward compat)
type (
	baseClient   = client.BaseClient
	HTTPClient   = client.HTTPClient
	BodyHandler  = client.BodyHandler
	FileResponse = client.FileResponse
)

var (
	NewBaseClient = client.NewBaseClient
	newBaseClient = client.NewBaseClient
)

// Error variable aliases
var (
	ErrUnauthenticated       = client.ErrUnauthenticated
	ErrPasswordResetRequired = client.ErrPasswordResetRequired
	ErrMissingLicense        = client.ErrMissingLicense
	ErrEndUserAuthRequired   = client.ErrEndUserAuthRequired
)

// Error type aliases
type (
	SetupAlreadyErr = client.SetupAlreadyErr
	NotFoundErr     = client.NotFoundErrIface
	ConflictErr     = client.ConflictErr
	notFoundErr     = client.NotFoundErr
	StatusCodeErr   = client.StatusCodeErr
)

func isNotFoundErr(err error) bool { return client.IsNotFoundErr(err) }
func IsNotFoundErr(err error) bool { return client.IsNotFoundErr(err) }

func extractServerErrorText(body io.Reader) string {
	return client.ExtractServerErrorText(body)
}

func extractServerErrorNameReason(body io.Reader) (string, string) {
	return client.ExtractServerErrorNameReason(body)
}

func extractServerErrorNameReasons(body io.Reader) ([]string, []string) {
	return client.ExtractServerErrorNameReasons(body)
}

// setupAlreadyErr and notSetupErr
type (
	setupAlreadyErr = client.SetupAlreadyError
	notSetupErr     = client.NotSetupError
	NotSetupErr     = client.NotSetupErr
)

// conflictErr
type conflictErr = client.ConflictError

// serverError is used for JSON parsing in extractServerErrMsg (in client_trigger.go).
type serverError struct {
	Message string `json:"message"`
	Errors  []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"errors"`
}
