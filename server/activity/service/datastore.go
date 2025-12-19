package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity"
)

// Datastore defines the datastore interface for the activity bounded context.
// This interface is internal to the activity context and should not be
// imported by other bounded contexts.
//
// Other bounded contexts should use the public service interface instead.
type Datastore interface {
	// Ping verifies database connectivity.
	Ping(ctx context.Context) error

	// ListActivities returns a paginated list of activities.
	ListActivities(ctx context.Context, opt activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error)

	// ListUsers returns a list of users matching the given options.
	// This is needed for activity search by user name/email.
	ListUsers(ctx context.Context, opt activity.UserListOptions) ([]*activity.User, error)
}
