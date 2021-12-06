package service

import (
	"context"
)

func (svc *Service) FlushSeenHosts(ctx context.Context) error {
	// No authorization check because this is used only internally.
	hostIDs := svc.seenHostSet.getAndClearHostIDs()
	return svc.ds.MarkHostsSeen(ctx, hostIDs, svc.clock.Now())
}
