package activity

import (
	"context"
)

// UpcomingActivityActivator activates the next upcoming activity in the queue.
// This is called when an activity implements ActivityActivator.
type UpcomingActivityActivator interface {
	ActivateNextUpcomingActivity(ctx context.Context, hostID uint, fromCompletedExecID string) error
}

// WebhookSendFunc is the function signature for sending a JSON payload to a URL.
type WebhookSendFunc = func(ctx context.Context, url string, payload any) error

// DataProviders combines all external dependency interfaces for the activity
// bounded context. The ACL adapter implements this single interface.
type DataProviders interface {
	UserProvider
	HostProvider
	AppConfigProvider
	UpcomingActivityActivator
}
