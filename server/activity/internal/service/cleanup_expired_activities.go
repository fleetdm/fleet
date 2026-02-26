package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// CleanupExpiredActivities deletes up to maxCount expired activities
// created before expiryThreshold that are not linked to any host.
func (s *Service) CleanupExpiredActivities(ctx context.Context, maxCount int, expiryThreshold time.Time) error {
	ctx, span := tracer.Start(ctx, "activity.service.CleanupExpiredActivities")
	defer span.End()

	if err := s.store.CleanupExpiredActivities(ctx, maxCount, expiryThreshold); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup expired activities")
	}
	return nil
}
