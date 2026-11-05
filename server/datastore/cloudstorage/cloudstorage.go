// Package cloudstorage holds error handling shared by the S3 and Azure Blob
// storage backends.
package cloudstorage

import (
	"errors"
	"fmt"
	"strings"
)

// MisconfiguredPrefix starts the message of every APIError that reports a
// misconfiguration.
const MisconfiguredPrefix = "Cloud storage is misconfigured. "

// APIError replaces the verbose vendor SDK message (request URL, raw body,
// request ID, ...) with a short one that keeps the vendor error code and
// message: "api error <code>: <message>".
//
// An error caused by how the store is configured (bad credentials, missing
// bucket or container, insufficient permissions, ...) also names the vendor
// and the resource, and starts with MisconfiguredPrefix.
//
// Unwrap returns a slice so that ctxerr.Cause stops at this error and the API
// response shows its message, while errors.As and errors.Is can still reach
// the vendor SDK error.
type APIError struct {
	vendor        string
	code          string
	message       string
	resource      string
	misconfigured bool
	err           error
}

// NewAPIError returns an APIError for the vendor error err. vendor names the
// backend, e.g. "S3" or "Azure". code and message come from the vendor error.
// resource identifies what was accessed, e.g. "bucket: my-bucket". Set
// misconfigured when the vendor error means the store is misconfigured.
func NewAPIError(err error, vendor, code, message, resource string, misconfigured bool) *APIError {
	// Vendors append request IDs and timestamps on extra lines.
	firstLine, _, _ := strings.Cut(message, "\n")
	return &APIError{
		vendor:        vendor,
		code:          code,
		message:       strings.TrimSpace(firstLine),
		resource:      resource,
		misconfigured: misconfigured,
		err:           err,
	}
}

// IsMisconfigured reports whether err is an APIError caused by the store's
// configuration.
func IsMisconfigured(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.misconfigured
}

func (e *APIError) Error() string {
	var b strings.Builder
	if e.misconfigured {
		b.WriteString(MisconfiguredPrefix)
		b.WriteString(e.vendor + ": ")
	}
	fmt.Fprintf(&b, "api error %s", e.code)
	if e.message != "" {
		b.WriteString(": " + e.message)
	}
	if e.misconfigured && e.resource != "" {
		fmt.Fprintf(&b, " (%s)", e.resource)
	}
	return b.String()
}

func (e *APIError) Unwrap() []error {
	return []error{e.err}
}
