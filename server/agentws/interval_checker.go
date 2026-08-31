package agentws

import (
	"context"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// DueLister lists which of the given hosts are due for a distributed read —
// interval work or an unanswered live query campaign — keyed by host ID with
// the reason it is due (see the fleet.AgentWSReason constants and
// fleet.AgentWSReasonLiveQuery). It is satisfied by fleet.Service.
type DueLister interface {
	ListHostIDsDueForDistributedRead(ctx context.Context, hostIDs []uint) (map[uint]string, error)
}

// chunkPacingDelay is the pause between notification batches so a large
// due-set (e.g. after downtime, when every host looks due at once) does not
// produce a thundering herd of distributed/read calls.
const chunkPacingDelay = 100 * time.Millisecond

// IntervalChecker is the per-instance job that periodically notifies connected
// agents with due work: interval work (labels, policies, host vitals) and
// unanswered live query campaigns whose one-shot pub/sub wake-up was missed.
// Each instance checks only the WebSocket connections it holds, so the work is
// naturally sharded with no coordination — intentionally NOT a cluster-wide
// locked cron job.
type IntervalChecker struct {
	Hub       *Hub
	Svc       DueLister
	Interval  time.Duration // how often to check (websocket.check_interval)
	BatchSize int           // hosts per due-check batch (websocket.check_batch_size)
	Logger    *slog.Logger
}

// Run checks held connections every Interval until ctx is done.
func (c *IntervalChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()
	c.Hub.RecordNextCheck(time.Now().Add(c.Interval))
	for {
		select {
		case <-ticker.C:
			c.Hub.RecordNextCheck(time.Now().Add(c.Interval))
			c.checkOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *IntervalChecker) checkOnce(ctx context.Context) {
	// A due host stays due until its results are ingested (or its live query
	// answered), so it is re-notified each tick until then — cheap, because
	// the agent coalesces triggers into one read+write iteration at a time
	// (see orbit/pkg/wstransport).
	hostIDs := c.Hub.HeldHostIDs()

	notified, disconnected := 0, 0
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
		// Group by reason so each notification carries what made its host due.
		byReason := make(map[string][]uint)
		for id, reason := range due {
			byReason[reason] = append(byReason[reason], id)
		}
		for reason, ids := range byReason {
			if reason == fleet.AgentWSReasonHostNotFound {
				// The host was deleted while its agent held a connection.
				// Nothing else closes it, so it would linger under the stale
				// host ID (and keep marking it seen) indefinitely.
				disconnected += c.Hub.Disconnect(ids)
				continue
			}
			notified += c.Hub.Notify(fleet.AgentWSMessageTypeDistributedRead, reason, ids)
		}
	}

	if notified > 0 || disconnected > 0 {
		c.Logger.DebugContext(ctx, "checked agents for due interval work",
			"checked", len(hostIDs), "notified", notified, "disconnected", disconnected)
	}
}
