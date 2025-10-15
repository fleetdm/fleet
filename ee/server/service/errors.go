package service

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type notFoundError struct {
	fleet.ErrorWithUUID
}

func (e notFoundError) Error() string {
	return "not found"
}

// IsNotFound implements the service.IsNotFound interface (from the non-premium
// service package) so that the handler returns 404 for this error.
func (e notFoundError) IsNotFound() bool {
	return true
}

type InvalidIDPTokenError struct{}

func (e InvalidIDPTokenError) Error() string {
	return "Invalid IDP token"
}

func (e InvalidIDPTokenError) StatusCode() int {
	return http.StatusForbidden
}

type InvalidCSRError struct{}

func (e InvalidCSRError) Error() string {
	return "Invalid CSR"
}

func (e InvalidCSRError) StatusCode() int {
	return http.StatusBadRequest
}
