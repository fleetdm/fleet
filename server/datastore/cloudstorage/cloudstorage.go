// Package cloudstorage holds error handling shared by the S3 and Azure Blob
// storage backends.
package cloudstorage

import (
	"fmt"
	"strings"
)

// MisconfiguredPrefix starts the message of every MisconfiguredError.
const MisconfiguredPrefix = "Cloud storage is misconfigured. "

// MisconfiguredError reports that the object store rejected a request because
// of how it is configured (bad credentials, missing bucket or container,
// insufficient permissions, ...). It replaces the verbose vendor SDK message
// with a short one that keeps the vendor error code and message.
//
// Unwrap returns a slice so that ctxerr.Cause stops at this error and the API
// response shows its message, while errors.As and errors.Is can still reach
// the vendor SDK error.
type MisconfiguredError struct {
	vendor   string
	code     string
	message  string
	resource string
	err      error
}

// NewMisconfigured returns a MisconfiguredError for the vendor error err.
// vendor names the backend, e.g. "S3" or "Azure". code and message come from
// the vendor error. resource identifies what was accessed, e.g.
// "bucket: my-bucket".
func NewMisconfigured(err error, vendor, code, message, resource string) *MisconfiguredError {
	// Vendors append request IDs and timestamps on extra lines.
	firstLine, _, _ := strings.Cut(message, "\n")
	return &MisconfiguredError{
		vendor:   vendor,
		code:     code,
		message:  strings.TrimSpace(firstLine),
		resource: resource,
		err:      err,
	}
}

func (e *MisconfiguredError) Error() string {
	var b strings.Builder
	b.WriteString(MisconfiguredPrefix)
	fmt.Fprintf(&b, "%s: api error %s", e.vendor, e.code)
	if e.message != "" {
		b.WriteString(": " + e.message)
	}
	if e.resource != "" {
		fmt.Fprintf(&b, " (%s)", e.resource)
	}
	return b.String()
}

func (e *MisconfiguredError) Unwrap() []error {
	return []error{e.err}
}
