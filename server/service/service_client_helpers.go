package service

// This file provides helpers used in server/service code that reference
// types/functions that have been moved to the client package.

import (
	"database/sql"
	"io"

	fleetclient "github.com/fleetdm/fleet/v4/client"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// extractServerErrorText extracts the error text from an HTTP response body.
// This delegates to the client package implementation.
func extractServerErrorText(body io.Reader) string {
	return fleetclient.ExtractServerErrorText(body)
}

// extractServerErrorNameReason extracts the error name and reason from an HTTP response body.
// This delegates to the client package implementation.
func extractServerErrorNameReason(body io.Reader) (string, string) {
	return fleetclient.ExtractServerErrorNameReason(body)
}

// extractServerErrorNameReasons extracts all error names and reasons from an HTTP response body.
// This delegates to the client package implementation.
func extractServerErrorNameReasons(body io.Reader) ([]string, []string) {
	return fleetclient.ExtractServerErrorNameReasons(body)
}

// notFoundErr is an error returned when a resource is not found.
// It is equivalent to the notFoundErr in the client package.
type notFoundErr struct {
	msg string

	fleet.ErrorWithUUID
}

func (e notFoundErr) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "The resource was not found"
}

func (e notFoundErr) NotFound() bool {
	return true
}

// Is allows errors.Is(err, sql.ErrNoRows) to return true for notFoundErr.
func (e notFoundErr) Is(other error) bool {
	return other == sql.ErrNoRows
}
