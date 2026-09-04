package apple_mdm

import (
	"context"
	"errors"
	"log/slog"
	mathrand "math/rand/v2"
	"sync"
	"time"

	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
)

const (
	// retryPendingCap bounds the in-memory pending set; on overflow the
	// oldest entry is dropped (the sweep covers it).
	retryPendingCap = 10_000
	// retryFlushBatch bounds how many due retries are re-pushed per flush.
	retryFlushBatch = 100
	// retryTick is how often the background loop looks for due retries.
	retryTick = 5 * time.Second
	// retryBreakerThreshold consecutive all-transient push calls — every
	// response failed transiently (APNs 5xx storm, GOAWAYs), or the call
	// itself failed — open the breaker for retryBreakerCooldown, pausing
	// the retry loop so retries don't pile onto a down APNs. Any success or
	// permanent rejection is a well-formed APNs answer and resets the
	// streak. Fresh enqueue-time pushes are never gated.
	retryBreakerThreshold = 5
	retryBreakerCooldown  = 2 * time.Minute
)

// retryBackoffs are the waits before each successive retry of an enrollment;
// after the last retry fails the entry is dropped and the daily sweep covers
// it.
var retryBackoffs = [...]time.Duration{30 * time.Second, 2 * time.Minute, 8 * time.Minute, 30 * time.Minute}

type retryEntry struct {
	// attempt counts retries scheduled so far; the wait behind nextAttempt
	// is retryBackoffs[attempt-1], and the entry is dropped once all of
	// retryBackoffs have been consumed.
	attempt     int
	nextAttempt time.Time
	queuedAt    time.Time
}

// RetryingPusher wraps a nanomdm push.Pusher and retries transient per-device
// push failures in-process with jittered backoff. Lossy by design: pushes are
// contentless and idempotent, and the APNs sweep bounds worst-case delivery,
// so a dropped retry (overflow, restart, giving up) only delays delivery,
// never loses commands. Push returns the inner results unchanged, so
// synchronous surfaces (lock, wipe, run-command) behave exactly as today.
type RetryingPusher struct {
	inner  nanomdm_push.Pusher
	logger *slog.Logger
	clock  func() time.Time

	mu                  sync.Mutex
	pending             map[string]*retryEntry
	consecutiveFailures int
	breakerOpenUntil    time.Time

	cancel context.CancelFunc
	done   chan struct{}
}

// NewRetryingPusher wraps inner and starts the background retry loop; call
// Stop to end it.
func NewRetryingPusher(inner nanomdm_push.Pusher, logger *slog.Logger) *RetryingPusher {
	p := newRetryingPusher(inner, logger)
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go p.loop(ctx)
	return p
}

// newRetryingPusher builds the wrapper without starting the loop, so tests
// can drive flushes deterministically through a fake clock.
func newRetryingPusher(inner nanomdm_push.Pusher, logger *slog.Logger) *RetryingPusher {
	return &RetryingPusher{
		inner:   inner,
		logger:  logger,
		clock:   time.Now,
		pending: make(map[string]*retryEntry),
		done:    make(chan struct{}),
	}
}

// Stop ends the background retry loop and waits for it to exit. Pending
// retries are dropped; the sweep covers them.
func (p *RetryingPusher) Stop() {
	if p.cancel != nil {
		p.cancel()
		<-p.done
	}
}

// Push sends through the wrapped pusher and returns its results and error
// unchanged (both can be non-nil at once: the push service returns partial
// results). Transient per-device failures are scheduled for background
// retry; permanent rejections never are — callers already route those (e.g.
// Unregistered to MDM turn-off).
func (p *RetryingPusher) Push(ctx context.Context, ids []string) (map[string]*nanomdm_push.Response, error) {
	res, err := p.inner.Push(ctx, ids)

	now := p.clock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.noteCallOutcomeLocked(ctx, now, res, err)
	for _, id := range ids {
		r, ok := res[id]
		switch {
		case !ok || r == nil:
			// no per-device verdict: a call-level failure (push cert or
			// push-info retrieval) leaves affected ids out of the map
			// entirely, and that's a transient failure for each of them. No
			// call-level error (e.g. the dev nop pusher) means nothing to do.
			if err != nil {
				p.scheduleRetryLocked(ctx, id, now)
			}
		case r.Err == nil:
			// an observed success supersedes any pending retry
			delete(p.pending, id)
		case !retryablePushErr(r.Err):
			// never retried; a pending entry is now known-hopeless too
			delete(p.pending, id)
		default:
			p.scheduleRetryLocked(ctx, id, now)
		}
	}
	return res, err
}

func (p *RetryingPusher) loop(ctx context.Context) {
	defer close(p.done)
	ticker := time.NewTicker(retryTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.flushDue(ctx)
		}
	}
}

// flushDue re-pushes up to retryFlushBatch due entries through the inner
// pusher and reschedules or drops them based on the outcome.
func (p *RetryingPusher) flushDue(ctx context.Context) {
	now := p.clock()
	p.mu.Lock()
	if now.Before(p.breakerOpenUntil) {
		p.mu.Unlock()
		return
	}
	var due []string
	for id, e := range p.pending {
		if !e.nextAttempt.After(now) {
			due = append(due, id)
			if len(due) >= retryFlushBatch {
				break
			}
		}
	}
	p.mu.Unlock()
	if len(due) == 0 {
		return
	}

	res, err := p.inner.Push(ctx, due)

	now = p.clock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.noteCallOutcomeLocked(ctx, now, res, err)
	// A foreground failure during the in-flight push can reset an entry's
	// ladder, which the stale result below re-escalates by one rung.
	// Tolerated: the window is milliseconds wide and the cost is one rung of
	// delay in a flow the sweep already bounds.
	for _, id := range due {
		r := res[id]
		if err != nil && r == nil {
			// call-level failure with no per-device verdict: not the
			// device's fault, keep the entry due; the breaker bounds how
			// hard a down APNs is re-attempted.
			continue
		}
		e := p.pending[id]
		if e == nil {
			continue
		}
		switch {
		case r == nil || r.Err == nil:
			delete(p.pending, id)
		case !retryablePushErr(r.Err):
			p.logger.InfoContext(ctx, "dropping APNs retry, failure is not retryable",
				"enrollment_id", id, "reason", APNSReason(r.Err), "err", r.Err)
			delete(p.pending, id)
		case e.attempt >= len(retryBackoffs):
			p.logger.InfoContext(ctx, "giving up on APNs retry, the sweep covers it",
				"enrollment_id", id, "attempts", e.attempt)
			delete(p.pending, id)
		default:
			e.nextAttempt = now.Add(jitteredBackoff(retryBackoffs[e.attempt]))
			e.attempt++
		}
	}
}

// noteCallOutcomeLocked feeds the circuit breaker with one inner.Push
// outcome. A call strikes when it failed outright or when every response
// failed transiently — the shapes an APNs outage produces, since the
// provider surfaces 5xx and GOAWAY as per-device errors with a nil
// call-level error. Any success or permanent rejection is a well-formed
// APNs answer and resets the streak. Requires p.mu held.
func (p *RetryingPusher) noteCallOutcomeLocked(ctx context.Context, now time.Time, res map[string]*nanomdm_push.Response, err error) {
	var transientFailures, healthySignals int
	for _, r := range res {
		switch {
		case r == nil || r.Err == nil:
			healthySignals++
		case retryablePushErr(r.Err):
			transientFailures++
		default:
			healthySignals++
		}
	}

	switch {
	case healthySignals > 0:
		p.consecutiveFailures = 0
	case err == nil && transientFailures == 0:
		// no verdicts at all (e.g. the dev nop pusher): not an APNs signal
	default:
		p.consecutiveFailures++
		if p.consecutiveFailures >= retryBreakerThreshold && !now.Before(p.breakerOpenUntil) {
			p.breakerOpenUntil = now.Add(retryBreakerCooldown)
			p.consecutiveFailures = 0
			p.logger.WarnContext(ctx, "pausing APNs retries, pushes are failing at the provider level",
				"cooldown", retryBreakerCooldown.String())
		}
	}
}

// scheduleRetryLocked adds id to the pending set at the first backoff step.
// Duplicates collapse onto one entry; a fresh enqueue-time failure restarts
// an entry that had already backed off past the first step, so a device
// failing right now isn't stuck waiting out an inherited 30m backoff.
// Requires p.mu held.
func (p *RetryingPusher) scheduleRetryLocked(ctx context.Context, id string, now time.Time) {
	if e, ok := p.pending[id]; ok {
		if e.attempt > 1 {
			e.attempt = 1
			e.nextAttempt = now.Add(jitteredBackoff(retryBackoffs[0]))
			e.queuedAt = now
		}
		return
	}
	if len(p.pending) >= retryPendingCap {
		var oldestID string
		var oldestQueuedAt time.Time
		for pid, e := range p.pending {
			if oldestID == "" || e.queuedAt.Before(oldestQueuedAt) {
				oldestID, oldestQueuedAt = pid, e.queuedAt
			}
		}
		delete(p.pending, oldestID)
		p.logger.WarnContext(ctx, "APNs retry set is full, dropping the oldest pending retry",
			"dropped_enrollment_id", oldestID, "cap", retryPendingCap)
	}
	p.pending[id] = &retryEntry{
		attempt:     1,
		nextAttempt: now.Add(jitteredBackoff(retryBackoffs[0])),
		queuedAt:    now,
	}
}

// retryablePushErr reports whether a per-device push error is worth
// retrying: transient APNs and transport failures are; permanent APNs
// rejections and enrollments the push service has no push info for
// (deleted or unknown) are not.
func retryablePushErr(err error) bool {
	return !isPermanentAPNSRejection(err) && !errors.Is(err, nanomdm_pushsvc.ErrIdNotFound)
}

// jitteredBackoff spreads d by ±20% so a burst of failures doesn't retry in
// lockstep.
func jitteredBackoff(d time.Duration) time.Duration {
	delta := (mathrand.Float64()*2 - 1) * 0.2 * float64(d) //nolint:gosec // non-security randomness
	return d + time.Duration(delta)
}
