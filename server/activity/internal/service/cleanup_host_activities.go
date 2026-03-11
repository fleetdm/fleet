package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// CleanupHostActivities removes activity_host_past rows for the given host IDs.
func (s *Service) CleanupHostActivities(ctx context.Context, hostIDs []uint) error {
	if err := s.store.CleanupHostActivities(ctx, hostIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup host activities")
	}
	return nil
}
