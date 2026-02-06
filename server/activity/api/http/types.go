// Package http provides HTTP request and response types for the activity bounded context.
// These types are used exclusively by the activities endpoint handler.
package http

import (
	"github.com/fleetdm/fleet/v4/server/activity/api"
)

// ListActivitiesRequest is the HTTP request type for listing activities.
type ListActivitiesRequest struct {
	ListOptions api.ListOptions `url:"list_options"`
}

// ListActivitiesResponse is the HTTP response type for listing activities.
type ListActivitiesResponse struct {
	Meta       *api.PaginationMetadata `json:"meta"`
	Activities []*api.Activity         `json:"activities"`
	Err        error                   `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r ListActivitiesResponse) Error() error { return r.Err }
