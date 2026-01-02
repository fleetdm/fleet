// Package types defines the internal types for the activity bounded context.
// This package has NO dependencies on other Fleet packages.
package types

import (
	"context"
	"encoding/json"
	"time"
)

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

// ListOptions defines pagination and sorting options for listing activities.
// Note: These fields are populated by the custom `url:"list_options"` handler in endpoint_utils.go,
// NOT by `query` struct tags. Activity-specific filters (ActivityType, etc.) are parsed separately
// from the request struct's own `query` tags and merged into this struct.
type ListOptions struct {
	// Pagination fields - parsed by listOptionsFromRequest from query params
	Page           uint
	PerPage        uint
	OrderKey       string
	OrderDirection string // "asc" or "desc"

	// Activity-specific filters - set from request struct after parsing
	ActivityType   string
	StartCreatedAt string // ISO date string
	EndCreatedAt   string // ISO date string
	MatchQuery     string // Search query for actor_full_name and actor_email

	// Internal fields - set programmatically, not from query params
	IncludeMetadata bool
	Streamed        *bool
	MatchingUserIDs []uint // User IDs matching MatchQuery (populated by service via ACL)
}

// PaginationMetadata contains pagination information.
type PaginationMetadata struct {
	HasNextResults     bool `json:"has_next_results"`
	HasPreviousResults bool `json:"has_previous_results"`
	TotalResults       uint `json:"total_results,omitempty"`
}

// Datastore is the datastore interface for the activity bounded context.
type Datastore interface {
	ListActivities(ctx context.Context, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}

// Service is the service interface for the activity bounded context.
type Service interface {
	ListActivities(ctx context.Context, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}
