package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
)

// The cap keeps the first run after an upgrade from deleting every expired notification in one run while the rest of the schedule waits.
const (
	endUserNotificationCleanupBatchSize = 1000
	endUserNotificationCleanupMaxRows   = 100_000
)

func (s *Service) CleanupNotifications(ctx context.Context) error {
	// one cutoff for the whole run, so every batch deletes against the same expiry
	olderThan := time.Now().UTC().Add(-api.EndUserNotificationRetention)

	var total int64
	for total < endUserNotificationCleanupMaxRows {
		deleted, err := s.ds.DeleteExpiredEndUserNotifications(ctx, olderThan, endUserNotificationCleanupBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired end user notifications")
		}
		total += deleted

		// a short batch is the end of what there was to delete
		if deleted < endUserNotificationCleanupBatchSize {
			break
		}
	}

	if total > 0 {
		s.logger.InfoContext(ctx, "deleted expired end user notifications", "count", total)
	}
	return nil
}
