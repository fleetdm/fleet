// Package backoff provides a stateful exponential backoff tracker for
// agent-to-server communication paths.
//
// Unlike pkg/retry (which blocks until success), this package tracks
// consecutive failures and computes the next polling interval so callers
// can integrate backoff into existing ticker-based loops.
//
// Each Tracker instance is independent -- one per communication path --
// so a failure on one path does not affect another.
package backoff

import (
	"math/rand/v2"
	"sync"
	"time"
)

// Tracker tracks consecutive failures for a single communication path
// and computes the next wait interval using exponential backoff with jitter.
//
// A Tracker is safe for concurrent use.
type Tracker struct {
	mu                  sync.Mutex
	baseInterval        time.Duration
	maxBackoff          time.Duration
	consecutiveFailures int
	inBackoff           bool
	backoffStartedAt    time.Time
}

// New creates a Tracker with the given base polling interval and maximum
// backoff ceiling.
func New(baseInterval, maxBackoff time.Duration) *Tracker {
	return &Tracker{
		baseInterval: baseInterval,
		maxBackoff:   maxBackoff,
	}
}

// RecordSuccess resets the backoff state. The next call to Interval
// returns baseInterval.
func (t *Tracker) RecordSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveFailures = 0
	t.inBackoff = false
	t.backoffStartedAt = time.Time{}
}

// RecordFailure records a failed request and advances the backoff.
func (t *Tracker) RecordFailure() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.inBackoff {
		t.inBackoff = true
		t.backoffStartedAt = time.Now()
	}
	t.consecutiveFailures++
}

// Interval returns the duration to wait before the next request.
// On success (consecutiveFailures == 0) this is baseInterval.
// On failure it is min(baseInterval * 2^consecutiveFailures + jitter, maxBackoff).
func (t *Tracker) Interval() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.consecutiveFailures == 0 {
		return t.baseInterval
	}

	// Calculate the exponential interval, clamping before overflow.
	// Once 2^shift * baseInterval exceeds maxBackoff the result is
	// clamped, so we can stop shifting early.
	interval := t.maxBackoff
	shift := t.consecutiveFailures
	if shift <= 62 {
		candidate := t.baseInterval
		for range shift {
			candidate *= 2
			if candidate >= t.maxBackoff || candidate <= 0 {
				candidate = t.maxBackoff
				break
			}
		}
		interval = candidate
	}

	// Add jitter: uniform random in [0, 10% of interval].
	tenthInterval := int64(interval / 10)
	if tenthInterval > 0 {
		jitter := time.Duration(rand.Int64N(tenthInterval))
		interval += jitter
	}

	if interval > t.maxBackoff {
		interval = t.maxBackoff
	}

	return interval
}

// InBackoff reports whether the tracker is currently in a backoff state
// (at least one consecutive failure with no intervening success).
func (t *Tracker) InBackoff() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.inBackoff
}

// ConsecutiveFailures returns the current consecutive failure count.
func (t *Tracker) ConsecutiveFailures() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.consecutiveFailures
}

// BackoffDuration returns how long the tracker has been in backoff,
// or 0 if not currently in backoff.
func (t *Tracker) BackoffDuration() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.inBackoff {
		return 0
	}
	return time.Since(t.backoffStartedAt)
}
