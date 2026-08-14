package api

import (
	"context"
	"time"
)

// DelayNotificationService puts a notification back in the queue for a later
// attempt. Kinds are implemented in server/service but don't own the
// notifications table, so a kind's OnDelay calls this rather than a
// datastore method directly.
type DelayNotificationService interface {
	DelayNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error
}
