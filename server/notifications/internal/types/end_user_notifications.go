// Package types defines internal interfaces for the notifications bounded
// context.
package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
)

// Datastore is the datastore interface for the notifications bounded context.
type Datastore interface {
	NewEndUserNotification(ctx context.Context, notification *api.EndUserNotification) (*api.EndUserNotification, error)
	GetEndUserNotificationByUUID(ctx context.Context, notificationUUID string) (*api.EndUserNotification, error)
	GetEndUserNotificationByExecutionID(ctx context.Context, executionID string) (*api.EndUserNotification, error)
	// ListEndUserNotificationsToDispatch returns the oldest due notifications,
	// at most one per host; limit bounds notifications read, not hosts
	// returned.
	ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*api.EndUserNotification, error)
	// SetEndUserNotificationsDispatched records the execution each
	// notification was queued as.
	SetEndUserNotificationsDispatched(ctx context.Context, notifications []*api.EndUserNotification) error
	// ExpireEndUserNotifications sets notifications past their expiry to
	// expired, and returns how many.
	ExpireEndUserNotifications(ctx context.Context) (int64, error)
	// VerifyEndUserNotification records the first time the notification
	// reached the end user; a later call doesn't move the timestamp.
	VerifyEndUserNotification(ctx context.Context, notificationUUID string, displayedAt time.Time) error
	// DelayEndUserNotification puts a notification back in the queue for a
	// later attempt, dropping the execution it was dispatched with.
	DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error
	// SetEndUserNotificationOutcome records how a display attempt ended.
	SetEndUserNotificationOutcome(ctx context.Context, notificationUUID string, outcome api.NotificationOutcome, nextAttemptAt *time.Time) error
}

// NotFoundError is returned when a notification can't be found by its
// identifier.
type NotFoundError struct {
	Identifier string
}

func (e *NotFoundError) Error() string {
	return "end user notification not found: " + e.Identifier
}

func (e *NotFoundError) IsNotFound() bool { return true }
