package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// CleanupExpiredActivities deletes up to maxCount activities older than expiryWindowDays
// that are not linked to any host.
func (s *Service) CleanupExpiredActivities(ctx context.Context, maxCount int, expiryWindowDays int) error {
	if err := s.store.CleanupExpiredActivities(ctx, maxCount, expiryWindowDays); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup expired activities")
	}
	return nil
}
