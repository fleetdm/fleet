package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// the cap keeps the first run after an upgrade from deleting everything at once while the rest of the schedule waits
const (
	endUserNotificationCleanupBatchSize = 1000
	endUserNotificationCleanupMaxRows   = 100_000
)

func (s *Service) CleanupNotifications(ctx context.Context, retention time.Duration) error {
	// one cutoff for the whole run, so every batch deletes against the same expiry
	olderThan := time.Now().UTC().Add(-retention)

	var total int64
	for total < endUserNotificationCleanupMaxRows {
		deleted, err := s.ds.DeleteExpiredEndUserNotifications(ctx, olderThan, endUserNotificationCleanupBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired end user notifications")
		}
		total += deleted

		// stop at a short batch, there is nothing left to delete
		if deleted < endUserNotificationCleanupBatchSize {
			break
		}
	}

	if total > 0 {
		s.logger.InfoContext(ctx, "deleted expired end user notifications", "count", total)
	}
	return nil
}
