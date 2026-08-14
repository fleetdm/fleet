package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// patchNotificationKind is the notify-before-patching kind. OnVerify and
// OnOutcome aren't implemented yet (#50912).
type patchNotificationKind struct {
	ds fleet.Datastore
}

func (k *patchNotificationKind) Name() string {
	return "notify_before_patching"
}

func (k *patchNotificationKind) OnVerify(ctx context.Context, notification *fleet.EndUserNotification, displayedAt time.Time) error {
	return nil
}

// 55 minutes from creation, not from now, so a delay rejoins the notification's
// 1-hour-then-5-minute schedule instead of starting a fresh wait. Falls back to
// a full delay from now if that mark has already passed.
func (k *patchNotificationKind) OnDelay(ctx context.Context, notification *fleet.EndUserNotification) error {
	now := time.Now().UTC()
	nextAttemptAt := notification.CreatedAt.Add(55 * time.Minute)
	if nextAttemptAt.Before(now) {
		nextAttemptAt = now.Add(fleet.EndUserNotificationDelayInterval)
	}
	return k.ds.DelayEndUserNotification(ctx, notification.UUID, nextAttemptAt)
}

func (k *patchNotificationKind) OnOutcome(ctx context.Context, notification *fleet.EndUserNotification, outcome fleet.NotificationOutcome) error {
	return nil
}
