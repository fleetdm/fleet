// Package http provides HTTP request and response types for the activity API.
package http

import (
	"github.com/fleetdm/fleet/v4/server/activity/api"
)

// ListActivitiesRequest is the HTTP request type for listing activities.
type ListActivitiesRequest struct {
	ListOptions    api.ListOptions `url:"list_options"`
	Query          string          `query:"query,optional"`
	ActivityType   string          `query:"activity_type,optional"`
	StartCreatedAt string          `query:"start_created_at,optional"`
	EndCreatedAt   string          `query:"end_created_at,optional"`
}

// ListActivitiesResponse is the HTTP response type for listing activities.
type ListActivitiesResponse struct {
	Meta       *api.PaginationMetadata `json:"meta"`
	Activities []*api.Activity         `json:"activities"`
	Err        error                   `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r ListActivitiesResponse) Error() error { return r.Err }
