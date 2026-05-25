// Package backoff provides a stateful exponential backoff tracker for
// agent-to-server communication paths.
//
// Unlike pkg/retry (which blocks until success), this package tracks
// consecutive failures and computes the next polling interval so callers
// can integrate backoff into existing ticker-based loops.
//
// Each Tracker instance is independent -- one per communication path --
// so a healthy path cannot reset the backoff of a failing one.
package backoff

import (
	"math/rand/v2"
	"sync"
	"time"
)

const (
	// maxShift caps the bit-shift exponent to prevent integer overflow
	// from wrapping into a small positive value that looks like a valid
	// interval.
	maxShift = 20

	// minInterval is the absolute floor returned by Interval to
	// guarantee callers never receive a zero or negative duration
	// (which would panic time.Ticker).
	minInterval = 1 * time.Second
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
// backoff ceiling. Both values are floored at minInterval (1s) to
// guarantee Interval never returns a value that would panic a ticker.
func New(baseInterval, maxBackoff time.Duration) *Tracker {
	if baseInterval < minInterval {
		baseInterval = minInterval
	}
	if maxBackoff < minInterval {
		maxBackoff = minInterval
	}
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
//
// When consecutiveFailures is 0 it returns baseInterval. Otherwise it
// returns min(baseInterval * 2^failures + jitter, maxBackoff).
func (t *Tracker) Interval() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.consecutiveFailures == 0 {
		return t.baseInterval
	}

	shift := min(t.consecutiveFailures, maxShift)
	interval := t.baseInterval << shift
	// Shift back to detect overflow: if we don't get the original
	// value, the left shift overflowed and the result is garbage.
	if interval>>shift != t.baseInterval {
		interval = t.maxBackoff
	}
	interval = min(interval, t.maxBackoff)
	interval += jitter(interval)
	interval = min(interval, t.maxBackoff)

	return interval
}

// jitter returns a random duration in [0, 10% of d).
func jitter(d time.Duration) time.Duration {
	tenth := int64(d / 10)
	if tenth <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(tenth)) //nolint:gosec // jitter does not need cryptographic randomness
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

// TimeSinceBackoffStarted returns how long ago the tracker entered backoff,
// or 0 if not currently in backoff.
func (t *Tracker) TimeSinceBackoffStarted() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.inBackoff {
		return 0
	}
	return time.Since(t.backoffStartedAt)
}
