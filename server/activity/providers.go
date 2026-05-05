package activity

import (
	"context"
)

// UpcomingActivityActivator activates the next upcoming activity in the queue.
// This is called when an activity implements ActivityActivator.
type UpcomingActivityActivator interface {
	ActivateNextUpcomingActivity(ctx context.Context, hostID uint, fromCompletedExecID string) error
}

// DataProviders combines all external dependency interfaces for the activity
// bounded context. The ACL adapter implements this single interface.
type DataProviders interface {
	UserProvider
	HostProvider
	AppConfigProvider
	UpcomingActivityActivator
}
