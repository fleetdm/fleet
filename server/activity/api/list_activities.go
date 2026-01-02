// Package api provides the public API for the activity bounded context.
// External code should use this package to interact with activities.
package api

import (
	"context"
	"encoding/json"
	"time"
)

// ListActivitiesService defines the contract for listing activities.
// Consumers should depend on this interface, not on internal implementations.
type ListActivitiesService interface {
	// ListActivities returns a paginated list of activities.
	// Authorization is checked internally.
	ListActivities(ctx context.Context, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}

// Service is the composite interface for the activity bounded context.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	ListActivitiesService
}

// Activity represents a recorded activity in the audit log.
type Activity struct {
	ID             uint             `json:"id,omitempty" db:"id"`
	UUID           string           `json:"uuid,omitempty" db:"uuid"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	Type           string           `json:"type" db:"activity_type"`
	ActorID        *uint            `json:"actor_id,omitempty" db:"user_id"`
	ActorFullName  *string          `json:"actor_full_name,omitempty" db:"name"`
	ActorEmail     *string          `json:"actor_email,omitempty" db:"user_email"`
	ActorGravatar  *string          `json:"actor_gravatar,omitempty" db:"gravatar_url"`
	ActorAPIOnly   *bool            `json:"actor_api_only,omitempty" db:"api_only"`
	Streamed       *bool            `json:"-" db:"streamed"`
	FleetInitiated bool             `json:"fleet_initiated" db:"fleet_initiated"`
	Details        *json.RawMessage `json:"details" db:"details"`
}

// ListOptions defines options for listing activities.
type ListOptions struct {
	// Pagination
	Page    uint
	PerPage uint
	After   string // Cursor-based pagination: start after this value (used with OrderKey)

	// Sorting
	OrderKey       string // Field to order by (e.g., "created_at", "id")
	OrderDirection string // "asc" or "desc"

	// Filters
	ActivityType   string // Filter by activity type
	StartCreatedAt string // ISO date string, filter activities created after this time
	EndCreatedAt   string // ISO date string, filter activities created before this time
	MatchQuery     string // Search query for actor name and email
	Streamed       *bool  // Filter by streamed status (nil = all, true = streamed only, false = not streamed only)
}

// PaginationMetadata contains pagination information for list responses.
type PaginationMetadata struct {
	HasNextResults     bool `json:"has_next_results"`
	HasPreviousResults bool `json:"has_previous_results"`
	TotalResults       uint `json:"total_results,omitempty"`
}
