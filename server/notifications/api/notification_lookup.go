package api

import "context"

// NotificationLookupService resolves which notification a queued script
// execution belongs to. server/service uses this to expand a notification's
// URL into the script it queued, without owning the notifications table itself.
type NotificationLookupService interface {
	// NotificationUUIDForExecution returns the UUID of the notification that
	// execution ID belongs to, or a not-found error if it isn't one.
	NotificationUUIDForExecution(ctx context.Context, executionID string) (string, error)
}
