// Package types defines internal interfaces for the notifications bounded
// context.
package types

import (
	"context"
	"encoding/json"
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
	DeferEndUserNotificationsForHosts(ctx context.Context, hostIDs []uint) error
	ExpireEndUserNotifications(ctx context.Context) (int64, error)
	VerifyEndUserNotification(ctx context.Context, notificationUUID string, displayedAt time.Time) error
	DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time, payload json.RawMessage) error
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

// InvalidArgumentError is what the response encoder answers with 422, the same
// way it reads IsNotFound above for a 404.
type InvalidArgumentError struct {
	Name   string
	Reason string
}

func (e *InvalidArgumentError) Error() string {
	return "validation failed: " + e.Name + " " + e.Reason
}

func (e *InvalidArgumentError) Invalid() []map[string]string {
	return []map[string]string{{"name": e.Name, "reason": e.Reason}}
}
