package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// DelayNotification puts a notification back in the queue for a later
// attempt. Called by a kind's OnDelay, since kinds don't own the
// notifications table directly.
func (s *Service) DelayNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error {
	if err := s.ds.DelayEndUserNotification(ctx, notificationUUID, nextAttemptAt); err != nil {
		return ctxerr.Wrap(ctx, err, "delay end user notification")
	}
	return nil
}
