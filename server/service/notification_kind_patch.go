package service

import (
	"context"
	"time"

	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
)

// patchNotificationKind is the notify-before-patching kind. OnVerify and
// OnOutcome aren't implemented yet (#50912).
type patchNotificationKind struct {
	delaySvc notifications_api.DelayNotificationService
}

// NewPatchNotificationKind creates the "patch" notification kind, for
// registration with the notifications bounded context.
func NewPatchNotificationKind(delaySvc notifications_api.DelayNotificationService) notifications_api.NotificationKind {
	return &patchNotificationKind{delaySvc: delaySvc}
}

func (k *patchNotificationKind) Name() string {
	return "patch"
}

func (k *patchNotificationKind) OnVerify(ctx context.Context, notification *notifications_api.EndUserNotification, displayedAt time.Time) error {
	return nil
}

// 55 minutes from when it was first displayed, not from now, so a delay
// rejoins the notification's 1-hour-then-5-minute schedule instead of
// starting a fresh wait. Falls back to a full delay from now if it was never
// displayed, or if that mark has already passed.
//
// TODO: should there be a grace period where we install no matter what?
func (k *patchNotificationKind) OnDelay(ctx context.Context, notification *notifications_api.EndUserNotification) error {
	now := time.Now().UTC()
	nextAttemptAt := now.Add(notifications_api.EndUserNotificationDelayInterval)
	if notification.DisplayedAt != nil {
		if fromDisplay := notification.DisplayedAt.Add(55 * time.Minute); fromDisplay.After(now) {
			nextAttemptAt = fromDisplay
		}
	}
	// nil payload: the patch kind doesn't change its wording yet, that comes
	// with the rest of the kind in #50912.
	return k.delaySvc.DelayNotification(ctx, notification.UUID, nextAttemptAt, nil)
}

func (k *patchNotificationKind) OnOutcome(ctx context.Context, notification *notifications_api.EndUserNotification, outcome notifications_api.NotificationOutcome) error {
	return nil
}
