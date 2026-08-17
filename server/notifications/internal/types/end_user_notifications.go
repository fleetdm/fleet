// Package types defines internal interfaces for the notifications bounded
// context.
package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
)

// Datastore is what the service needs of the notifications tables. Each method
// is documented on its implementation in internal/mysql.
type Datastore interface {
	GetEndUserNotificationByUUID(ctx context.Context, notificationUUID string) (*api.EndUserNotification, error)
	GetEndUserNotificationByExecutionID(ctx context.Context, executionID string) (*api.EndUserNotification, error)
	ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*api.EndUserNotification, error)
	SetEndUserNotificationsDispatched(ctx context.Context, notifications []*api.EndUserNotification) error
	ExpireEndUserNotifications(ctx context.Context) (int64, error)
	VerifyEndUserNotification(ctx context.Context, notificationUUID string, displayedAt time.Time) error
	DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error
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
