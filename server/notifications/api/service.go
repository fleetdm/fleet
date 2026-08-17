// Package api provides the public API for the notifications bounded context.
package api

import (
	"context"
	"time"
)

// Service is everything the notifications bounded context can do, and what
// bootstrap returns. Some methods are also grouped into their own interfaces
// below, for callers that should only see those.
type Service interface {
	RecordOutcomeService
	NotificationLookupService
	DelayNotificationService

	// ExpireAndQueueNotifications gives up on notifications that are out of
	// time, then queues a script for each one that is due.
	ExpireAndQueueNotifications(ctx context.Context) error

	// GetNotificationForHost returns a not-found error if the notification
	// doesn't exist or belongs to a host other than hostID.
	GetNotificationForHost(ctx context.Context, hostID uint, notificationUUID string) (*EndUserNotification, error)

	// ApplyAction carries out what an end user chose to do with one of the
	// notifications on their host.
	ApplyAction(ctx context.Context, hostID uint, notificationUUID string, action EndUserNotificationAction) error

	// RegisterKind adds a kind of end user notification. Call it before the
	// server starts serving requests, since the registry isn't synchronized.
	RegisterKind(kind NotificationKind)
}

// RecordOutcomeService records how a queued script attempt ended. An execution
// that doesn't belong to a notification is a no-op rather than an error, since
// most script results have nothing to do with notifications.
type RecordOutcomeService interface {
	RecordOutcome(ctx context.Context, executionID string, exitCode int64, output string) error
}

// NotificationLookupService resolves which notification a queued script
// execution belongs to, so callers can build its URL without reading the
// notifications table themselves.
type NotificationLookupService interface {
	NotificationUUIDForExecution(ctx context.Context, executionID string) (string, error)
}

// DelayNotificationService puts a notification back in the queue for a later
// attempt. Kinds live outside this context and don't own the table, so their
// OnDelay calls this instead of a datastore method.
type DelayNotificationService interface {
	DelayNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error
}
