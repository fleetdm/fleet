package agentws

import (
	"context"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// DueLister lists which of the given hosts are due for a distributed read.
// It is satisfied by fleet.Service.
type DueLister interface {
	ListHostIDsDueForDistributedRead(ctx context.Context, hostIDs []uint) ([]uint, error)
}

// chunkPacingDelay is the pause between notification batches so a large
// due-set (e.g. after downtime, when every host looks due at once) does not
// produce a thundering herd of distributed/read calls.
const chunkPacingDelay = 100 * time.Millisecond

// IntervalChecker is the per-instance job that periodically notifies connected
// agents with due interval work (labels, policies, host vitals). Each server
// instance runs its own checker over only the WebSocket connections it holds,
// so the work is naturally sharded across instances with no coordination —
// this is intentionally NOT a cluster-wide locked cron job.
//
// In steady state hosts' updated-at timestamps are naturally spread out, so
// each tick finds only a small slice due. The one case where everything looks
// due at once (after downtime) is smoothed by batch pacing.
type IntervalChecker struct {
	Hub       *Hub
	Svc       DueLister
	Interval  time.Duration // how often to check (websocket.check_interval)
	Grace     time.Duration // re-notify grace period (websocket.renotify_grace_period)
	BatchSize int           // hosts per due-check batch (websocket.check_batch_size)
	Logger    *slog.Logger
}

// Run checks held connections every Interval until ctx is done.
func (c *IntervalChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.checkOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *IntervalChecker) checkOnce(ctx context.Context) {
	// Skip hosts notified within the grace period: a due host stays due until
	// its results are ingested, and we don't want to hammer slow responders on
	// every tick. Live query notifications share the same timestamps.
	hostIDs := c.Hub.FilterDueForRenotify(c.Hub.HeldHostIDs(), c.Grace)

	notified := 0
	for start := 0; start < len(hostIDs); start += c.BatchSize {
		if ctx.Err() != nil {
			return
		}
		if start > 0 {
			// Pace batches to spread the resulting distributed/read load.
			select {
			case <-time.After(chunkPacingDelay):
			case <-ctx.Done():
				return
			}
		}

		end := min(start+c.BatchSize, len(hostIDs))
		due, err := c.Svc.ListHostIDsDueForDistributedRead(ctx, hostIDs[start:end])
		if err != nil {
			c.Logger.ErrorContext(ctx, "list hosts due for distributed read", "err", err)
			continue
		}
		notified += c.Hub.Notify(fleet.AgentWSMessageTypeDistributedRead, due)
	}

	if notified > 0 {
		c.Logger.DebugContext(ctx, "notified agents with due interval work",
			"checked", len(hostIDs), "notified", notified)
	}
}
