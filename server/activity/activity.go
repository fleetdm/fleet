package activity

import (
	"context"
	"encoding/json"
	"time"
)

// OrderDirection constants for list queries.
const (
	OrderAscending  = "asc"
	OrderDescending = "desc"
)

// ListOptions defines common options for paginated list queries.
type ListOptions struct {
	Page            uint   `query:"page,optional"`
	PerPage         uint   `query:"per_page,optional"`
	OrderKey        string `query:"order_key,optional"`
	OrderDirection  string `query:"order_direction,optional"`
	MatchQuery      string `query:"query,optional"`
	After           string `query:"after,optional"`
	IncludeMetadata bool
}

// UsesCursorPagination returns true if cursor-based pagination is being used.
func (l ListOptions) UsesCursorPagination() bool {
	return l.After != ""
}

// ListActivitiesOptions defines options for listing activities.
type ListActivitiesOptions struct {
	ListOptions
	Streamed       *bool
	ActivityType   string
	StartCreatedAt string
	EndCreatedAt   string
}

// PaginationMetadata contains pagination information for list responses.
type PaginationMetadata struct {
	HasNextResults     bool `json:"has_next_results"`
	HasPreviousResults bool `json:"has_previous_results"`
	TotalResults       uint `json:"total_results,omitempty"`
}

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

// AuthzType implements AuthzTyper for authorization checks.
func (*Activity) AuthzType() string {
	return "activity"
}

// ActionRead is the action for reading activities.
const ActionRead = "read"

// User represents minimal user information for activity enrichment.
type User struct {
	ID          uint   `db:"id"`
	Name        string `db:"name"`
	Email       string `db:"email"`
	GravatarURL string `db:"gravatar_url"`
	APIOnly     bool   `db:"api_only"`
}

// UserListOptions defines options for listing users.
type UserListOptions struct {
	ListOptions
}

// Service defines the public interface for the activity bounded context.
// Other bounded contexts should use this interface to interact with activities.
type Service interface {
	// Ping verifies the service is healthy.
	Ping(ctx context.Context) error

	// ListActivities returns a paginated list of activities for the organization.
	ListActivities(ctx context.Context, opt ListActivitiesOptions) ([]*Activity, *PaginationMetadata, error)
}

// Authorizer defines the authorization interface needed by the activity service.
type Authorizer interface {
	// SkipAuthorization marks the request as not requiring authorization.
	SkipAuthorization(ctx context.Context)
	// Authorize checks authorization for the provided object and action.
	Authorize(ctx context.Context, object, action any) error
}

// /////////////////////////////////////////////
// Activity API request and response structs

// DefaultResponse is the base response type for activity endpoints.
type DefaultResponse struct {
	Err error `json:"error,omitempty"`
}

// Error implements the Errorer interface.
func (r DefaultResponse) Error() error { return r.Err }

// PingResponse is the response for the ping endpoint.
type PingResponse struct {
	Message string `json:"message"`
	DefaultResponse
}

// ListActivitiesResponse is the response for the list activities endpoint.
type ListActivitiesResponse struct {
	Meta       *PaginationMetadata `json:"meta"`
	Activities []*Activity         `json:"activities"`
	DefaultResponse
}
