package mysql

import (
	"context"
	"fmt"
	"time"
)

type expiredHostsSeenTimeRange struct {
	start    time.Time
	end      time.Time
	workerID int
}

type expiredHostsSeenTimeRangeContextKey struct{}

// WithExpiredHostsSeenTimeRange limits CleanupExpiredHostsBatch to OSQuery-seen hosts
// whose seen_time falls in [start, end). Unseen and MDM-only hosts are left for
// the unscoped cleanup pass after all time buckets finish. workerID labels progress logs.
func WithExpiredHostsSeenTimeRange(ctx context.Context, start, end time.Time, workerID int) (context.Context, error) {
	if !start.Before(end) {
		return nil, fmt.Errorf("expired-host seen-time range start %s must be before end %s", start, end)
	}
	if workerID <= 0 {
		return nil, fmt.Errorf("expired-host worker ID must be positive, got %d", workerID)
	}
	return context.WithValue(ctx, expiredHostsSeenTimeRangeContextKey{}, expiredHostsSeenTimeRange{start: start, end: end, workerID: workerID}), nil
}
