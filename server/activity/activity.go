// Package activity defines the public types and interfaces for the activity bounded context.
// This package should have minimal dependencies and define its own types rather than
// importing from fleet package.
package activity

import (
	"context"
	"encoding/json"
	"time"
)

// ContextKey is a type for context keys used by the activity package.
type ContextKey string

// WebhookContextKey is the context key to indicate that the activity webhook has been processed.
const WebhookContextKey = ContextKey("ActivityWebhook")

// //////////////////////////////////////////
// Core types

// Activity represents a past activity in the audit log.
type Activity struct {
	ID             uint             `json:"id,omitempty" db:"id"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	ActorFullName  *string          `json:"actor_full_name,omitempty" db:"name"`
	ActorID        *uint            `json:"actor_id,omitempty" db:"user_id"`
	ActorGravatar  *string          `json:"actor_gravatar,omitempty" db:"gravatar_url"`
	ActorEmail     *string          `json:"actor_email,omitempty" db:"user_email"`
	ActorAPIOnly   *bool            `json:"actor_api_only,omitempty" db:"api_only"`
	Type           string           `json:"type" db:"activity_type"`
	Details        *json.RawMessage `json:"details" db:"details"`
	Streamed       *bool            `json:"-" db:"streamed"`
	FleetInitiated bool             `json:"fleet_initiated" db:"fleet_initiated"`
}

// AuthzType implements AuthzTyper to be able to verify access to activities.
func (*Activity) AuthzType() string {
	return "activity"
}

// //////////////////////////////////////////
// Activity details interfaces
// These interfaces define the contract for activity details types.
// The actual activity type implementations remain in the fleet package
// since they are domain-specific (e.g., ActivityTypeCreatedPack, etc.)

// Details is the interface that all activity detail types must implement.
type Details interface {
	// ActivityName returns the name/type of the activity.
	ActivityName() string
}

// DetailsWithHosts is implemented by activities related to specific hosts.
type DetailsWithHosts interface {
	Details
	HostIDs() []uint
}

// AutomatableDetails is implemented by activities that can be automated.
type AutomatableDetails interface {
	Details
	WasFromAutomation() bool
}

// HostOnlyDetails is implemented by activities that are host-only (not shown in org-wide list).
type HostOnlyDetails interface {
	Details
	HostOnly() bool
}

// //////////////////////////////////////////
// Actor information

// Actor represents the user who performed an activity.
// This is a minimal interface to avoid coupling to the fleet.User type.
type Actor struct {
	ID      uint
	Name    string
	Email   string
	Deleted bool
}

// //////////////////////////////////////////
// Pagination types

// OrderDirection defines the order direction for sorting.
type OrderDirection int

const (
	// OrderAscending sorts in ascending order.
	OrderAscending OrderDirection = iota
	// OrderDescending sorts in descending order.
	OrderDescending
)

// ListOptions defines pagination and sorting options.
type ListOptions struct {
	Page                        uint           `query:"page,optional"`
	PerPage                     uint           `query:"per_page,optional"`
	OrderKey                    string         `query:"order_key,optional"`
	OrderDirection              OrderDirection `query:"order_direction,optional"`
	MatchQuery                  string         `query:"query,optional"`
	After                       string         `query:"after,optional"`
	IncludeMetadata             bool
	TestSecondaryOrderKey       string
	TestSecondaryOrderDirection OrderDirection
}

// UsesCursorPagination returns true if cursor pagination is being used.
func (l ListOptions) UsesCursorPagination() bool {
	return l.After != ""
}

// PaginationMetadata contains pagination information for list responses.
type PaginationMetadata struct {
	HasNextResults     bool `json:"has_next_results"`
	HasPreviousResults bool `json:"has_previous_results"`
	TotalResults       uint `json:"-"`
}

// ListActivitiesOptions defines filtering options for listing activities.
type ListActivitiesOptions struct {
	ListOptions
	ActivityType   string `query:"activity_type,optional"`
	StartCreatedAt string `query:"start_created_at,optional"`
	EndCreatedAt   string `query:"end_created_at,optional"`
	Streamed       *bool
}

// //////////////////////////////////////////
// Service interface

// Service defines the public interface for the activity bounded context.
// Other bounded contexts should use this interface to interact with activities.
type Service interface {
	// Ping verifies the service is healthy.
	Ping(ctx context.Context) error

	// ListActivities returns activities matching the given options.
	ListActivities(ctx context.Context, opt ListActivitiesOptions) ([]*Activity, *PaginationMetadata, error)

	// ListHostPastActivities returns past activities for a specific host.
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)

	// NewActivity records a new activity in the audit log.
	NewActivity(ctx context.Context, actor *Actor, details Details, detailsJSON []byte, createdAt time.Time) error

	// MarkActivitiesAsStreamed marks activities as streamed to external destinations.
	MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error

	// CleanupActivitiesAndAssociatedData removes old activities.
	CleanupActivitiesAndAssociatedData(ctx context.Context, maxCount int, expiryWindowDays int) error
}

// //////////////////////////////////////////
// API request and response structs

// DefaultResponse is the base response type for activity endpoints.
type DefaultResponse struct {
	Err error `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r DefaultResponse) Error() error { return r.Err }

// PingResponse is the response for the ping endpoint.
type PingResponse struct {
	Message string `json:"message"`
	DefaultResponse
}

// ListActivitiesResponse is the response for listing activities.
type ListActivitiesResponse struct {
	Meta       *PaginationMetadata `json:"meta"`
	Activities []*Activity         `json:"activities"`
	DefaultResponse
}

// ListHostPastActivitiesResponse is the response for listing host past activities.
type ListHostPastActivitiesResponse struct {
	Meta       *PaginationMetadata `json:"meta"`
	Activities []*Activity         `json:"activities"`
	DefaultResponse
}
