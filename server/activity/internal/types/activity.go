// Package types defines internal interfaces for the activity bounded context.
package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity/api"
)

// activityWebhookContextKeyType is the context key type used to indicate that the activity webhook
// has been processed. This is a sanity check to ensure callers use the service layer
// (which handles webhooks) rather than calling the datastore directly.
type activityWebhookContextKeyType struct{}

// ActivityWebhookContextKey is used to mark that the webhook was processed before storing the activity.
var ActivityWebhookContextKey = activityWebhookContextKeyType{}

// ActivityAutomationAuthor is the name used for the actor when an activity
// is recorded as a result of an automated action (cron job, webhook, etc.)
// or policy automation (i.e. triggered by a failing policy).
const ActivityAutomationAuthor = "Fleet"

// AutomatableActivity indicates the activity was initiated by automation.
type AutomatableActivity interface {
	WasFromAutomation() bool
}

// ActivityHosts indicates the activity is associated with specific hosts.
type ActivityHosts interface {
	HostIDs() []uint
}

// ActivityHostOnly indicates the activity is host-scoped only.
type ActivityHostOnly interface {
	HostOnly() bool
}

// ActivityActivator indicates the activity should activate the next upcoming activity.
type ActivityActivator interface {
	MustActivateNextUpcomingActivity() bool
	ActivateNextUpcomingActivityArgs() (hostID uint, cmdUUID string)
}

// ListOptions extends api.ListOptions with internal fields used by the datastore.
type ListOptions struct {
	api.ListOptions

	// Internal fields: set programmatically by service, not from query params
	IncludeMetadata bool
	MatchingUserIDs []uint // User IDs matching MatchQuery (populated by service via ACL)
}

// GetPage returns the page number for pagination.
func (o *ListOptions) GetPage() uint { return o.Page }

// GetPerPage returns the number of items per page.
func (o *ListOptions) GetPerPage() uint { return o.PerPage }

// GetOrderKey returns the field to order by.
func (o *ListOptions) GetOrderKey() string { return o.OrderKey }

// IsDescending returns true if the order direction is descending.
func (o *ListOptions) IsDescending() bool { return o.OrderDirection == api.OrderDescending }

// GetCursorValue returns the cursor value for cursor-based pagination.
func (o *ListOptions) GetCursorValue() string { return o.After }

// WantsPaginationInfo returns true if pagination metadata should be included.
func (o *ListOptions) WantsPaginationInfo() bool { return o.IncludeMetadata }

// GetSecondaryOrderKey returns the secondary order key (not used for activities).
func (o *ListOptions) GetSecondaryOrderKey() string { return "" }

// IsSecondaryDescending returns true if the secondary order is descending (not used for activities).
func (o *ListOptions) IsSecondaryDescending() bool { return false }

// Datastore is the datastore interface for the activity bounded context.
type Datastore interface {
	ListActivities(ctx context.Context, opt ListOptions) ([]*api.Activity, *api.PaginationMetadata, error)
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*api.Activity, *api.PaginationMetadata, error)
	MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error
	// NewActivity stores a new activity record in the database.
	// The webhook context key must be set in the context before calling this method.
	NewActivity(ctx context.Context, user *api.User, activity api.ActivityDetails, details []byte, createdAt time.Time) error
}
