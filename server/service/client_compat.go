package service

import (
	"io"

	"github.com/fleetdm/fleet/v4/client"
)

// Type aliases for base client types (backward compat)
type baseClient = client.BaseClient
type HTTPClient = client.HTTPClient
type httpClient = client.HTTPClient
type bodyHandler = client.BodyHandler
type BodyHandler = client.BodyHandler
type FileResponse = client.FileResponse

var NewBaseClient = client.NewBaseClient
var newBaseClient = client.NewBaseClient
var errInvalidScheme = client.ErrInvalidScheme

// Error variable aliases
var (
	ErrUnauthenticated       = client.ErrUnauthenticated
	ErrPasswordResetRequired = client.ErrPasswordResetRequired
	ErrMissingLicense        = client.ErrMissingLicense
	ErrEndUserAuthRequired   = client.ErrEndUserAuthRequired
)

// Error type aliases
type SetupAlreadyErr = client.SetupAlreadyErr
type NotFoundErr = client.NotFoundErrIface
type ConflictErr = client.ConflictErr
type notFoundErr = client.NotFoundErr
type statusCodeErr = client.StatusCodeErr
type StatusCodeErr = client.StatusCodeErr

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

func truncateAndDetectHTML(body []byte, maxLen int) ([]byte, bool) {
	return client.TruncateAndDetectHTML(body, maxLen)
}

// setupAlreadyErr and notSetupErr
type setupAlreadyErr = client.SetupAlreadyError
type notSetupErr = client.NotSetupError
type NotSetupErr = client.NotSetupErr

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
