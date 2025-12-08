package service

import (
	"context"
	"time"

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

	// NewActivity stores a new past activity record.
	NewActivity(ctx context.Context, actor *activity.Actor, details activity.Details,
		detailsJSON []byte, createdAt time.Time) error

	// ListActivities returns activities matching the given options.
	ListActivities(ctx context.Context, opt activity.ListActivitiesOptions) (
		[]*activity.Activity, *activity.PaginationMetadata, error)

	// ListHostPastActivities returns past activities for a specific host.
	ListHostPastActivities(ctx context.Context, hostID uint, opt activity.ListOptions) (
		[]*activity.Activity, *activity.PaginationMetadata, error)

	// MarkActivitiesAsStreamed marks the specified activities as streamed.
	MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error

	// CleanupActivitiesAndAssociatedData removes old activities and related data.
	CleanupActivitiesAndAssociatedData(ctx context.Context, maxCount int, expiryWindowDays int) error
}
