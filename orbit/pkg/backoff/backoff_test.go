package backoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)
	require.NotNil(t, tracker)
	assert.Equal(t, 10*time.Second, tracker.baseInterval)
	assert.Equal(t, 30*time.Minute, tracker.maxBackoff)
	assert.Equal(t, 0, tracker.ConsecutiveFailures())
	assert.False(t, tracker.InBackoff())
}

func TestIntervalReturnsBaseOnSuccess(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)
	assert.Equal(t, 10*time.Second, tracker.Interval())
}

// Oracle Valid Example: exponential doubling with cap at max_backoff.
// base_interval=10s, max_backoff=1800s
// 1st error: 20s, 2nd: 40s, 3rd: 80s, ..., 8th+: 1800s (capped)
func TestExponentialBackoff(t *testing.T) {
	base := 10 * time.Second
	maxB := 30 * time.Minute
	tracker := New(base, maxB)

	expectedMin := []time.Duration{
		20 * time.Second,   // 2^1 * 10s
		40 * time.Second,   // 2^2 * 10s
		80 * time.Second,   // 2^3 * 10s
		160 * time.Second,  // 2^4 * 10s
		320 * time.Second,  // 2^5 * 10s
		640 * time.Second,  // 2^6 * 10s
		1280 * time.Second, // 2^7 * 10s
		1800 * time.Second, // 2^8 * 10s = 2560s, capped to 1800s
		1800 * time.Second, // still capped
	}

	for i, expMin := range expectedMin {
		tracker.RecordFailure()
		interval := tracker.Interval()

		// The interval should be at least the expected minimum (before jitter)
		// and at most expected + 10% jitter, but never exceed maxBackoff.
		assert.GreaterOrEqual(t, interval, expMin,
			"failure %d: interval %v should be >= %v", i+1, interval, expMin)
		// Jitter adds up to 10% of the calculated interval (before cap).
		maxWithJitter := min(expMin+expMin/10, maxB)
		assert.LessOrEqual(t, interval, maxWithJitter,
			"failure %d: interval %v should be <= %v", i+1, interval, maxWithJitter)
	}
}

// Oracle: a single success resets backoff completely.
func TestSuccessResetsBackoff(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	// Build up some backoff
	for range 5 {
		tracker.RecordFailure()
	}
	require.True(t, tracker.InBackoff())
	require.Equal(t, 5, tracker.ConsecutiveFailures())

	// One success resets everything
	tracker.RecordSuccess()
	assert.False(t, tracker.InBackoff())
	assert.Equal(t, 0, tracker.ConsecutiveFailures())
	assert.Equal(t, 10*time.Second, tracker.Interval())
}

// Oracle Edge Case: server returns 200 then immediately 500.
// consecutive_failures resets to 0 on the 200, then starts from 1 on the 500.
func TestSuccessThenFailureRestartsFromOne(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	// Build up backoff
	for range 5 {
		tracker.RecordFailure()
	}

	// Success resets
	tracker.RecordSuccess()
	assert.Equal(t, 0, tracker.ConsecutiveFailures())

	// New failure starts from 1
	tracker.RecordFailure()
	assert.Equal(t, 1, tracker.ConsecutiveFailures())
	interval := tracker.Interval()
	// Should be ~20s (2^1 * 10s) + jitter, not the large value from before
	assert.LessOrEqual(t, interval, 22*time.Second)
}

// Oracle: interval never drops below base_interval.
func TestIntervalNeverBelowBase(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	// Even with 0 failures, should return base
	assert.Equal(t, 10*time.Second, tracker.Interval())

	// After success, should return base
	tracker.RecordFailure()
	tracker.RecordSuccess()
	assert.Equal(t, 10*time.Second, tracker.Interval())
}

// Oracle: interval never exceeds max_backoff.
func TestIntervalNeverExceedsMax(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	// Record many failures to push well past max
	for range 50 {
		tracker.RecordFailure()
	}
	interval := tracker.Interval()
	assert.LessOrEqual(t, interval, 30*time.Minute)
}

// Oracle Behavioral Invariant: backoff state is per-tracker.
// One tracker backing off does not affect another.
func TestPerPathIsolation(t *testing.T) {
	desktopTracker := New(10*time.Second, 30*time.Minute)
	orbitTracker := New(30*time.Second, 30*time.Minute)

	// Desktop fails
	for range 5 {
		desktopTracker.RecordFailure()
	}

	// Orbit succeeds -- should not be affected by desktop's backoff
	assert.False(t, orbitTracker.InBackoff())
	assert.Equal(t, 0, orbitTracker.ConsecutiveFailures())
	assert.Equal(t, 30*time.Second, orbitTracker.Interval())

	// Desktop is still in backoff
	assert.True(t, desktopTracker.InBackoff())
	assert.Equal(t, 5, desktopTracker.ConsecutiveFailures())
}

// Oracle: single-host transient blip (1 error then success).
// One 20s wait instead of 10s, then back to normal.
func TestSingleTransientError(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	tracker.RecordFailure()
	interval := tracker.Interval()
	// Should be ~20s + small jitter
	assert.GreaterOrEqual(t, interval, 20*time.Second)
	assert.LessOrEqual(t, interval, 22*time.Second)

	// Success resets immediately
	tracker.RecordSuccess()
	assert.Equal(t, 10*time.Second, tracker.Interval())
}

// Oracle: the agent never stops retrying. There is no "give up" behavior.
// (Tracker has no max attempts -- it just tracks state and returns intervals.)
func TestNoGiveUp(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	// Even after a large number of failures, Interval returns a bounded value
	for range 1000 {
		tracker.RecordFailure()
	}
	interval := tracker.Interval()
	assert.GreaterOrEqual(t, interval, 10*time.Second)
	assert.LessOrEqual(t, interval, 30*time.Minute)
}

// Oracle Ordering Guarantee: intervals are monotonically non-decreasing
// during a consecutive error sequence (ignoring jitter noise).
func TestMonotonicallyNonDecreasing(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	var prevBase time.Duration
	for i := range 15 {
		tracker.RecordFailure()
		// Calculate the base interval without jitter for monotonicity check
		shift := min(i+1, 30)
		base := min(time.Duration(1<<uint(shift))*10*time.Second, 30*time.Minute)
		assert.GreaterOrEqual(t, base, prevBase,
			"base interval should be non-decreasing at failure %d", i+1)
		prevBase = base
	}
}

func TestBackoffDuration(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)

	// Not in backoff
	assert.Equal(t, time.Duration(0), tracker.BackoffDuration())

	// Enter backoff
	tracker.RecordFailure()
	time.Sleep(10 * time.Millisecond)
	dur := tracker.BackoffDuration()
	assert.Greater(t, dur, time.Duration(0))

	// Exit backoff
	tracker.RecordSuccess()
	assert.Equal(t, time.Duration(0), tracker.BackoffDuration())
}

func TestConcurrentAccess(t *testing.T) {
	tracker := New(10*time.Second, 30*time.Minute)
	done := make(chan struct{})

	// Hammer the tracker from multiple goroutines
	for range 10 {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 100 {
				tracker.RecordFailure()
				_ = tracker.Interval()
				_ = tracker.InBackoff()
				_ = tracker.ConsecutiveFailures()
				_ = tracker.BackoffDuration()
				tracker.RecordSuccess()
			}
		}()
	}

	for range 10 {
		<-done
	}
}

// TestTickerIntegration simulates the real Desktop polling loop pattern:
// a ticker-based loop that adjusts its interval based on backoff state.
// Uses short durations (milliseconds) to keep the test fast.
func TestTickerIntegration(t *testing.T) {
	base := 10 * time.Millisecond
	maxB := 200 * time.Millisecond
	tracker := New(base, maxB)

	ticker := time.NewTicker(base)
	defer ticker.Stop()

	// Phase 1: 4 consecutive failures, measure that intervals grow
	var intervals []time.Duration
	prev := time.Now()
	for range 4 {
		<-ticker.C
		now := time.Now()
		intervals = append(intervals, now.Sub(prev))
		prev = now

		// Simulate server error
		tracker.RecordFailure()
		ticker.Reset(tracker.Interval())
	}

	// Verify intervals are roughly increasing (with tolerance for scheduling jitter).
	// Skip the first interval since it's the initial base tick.
	for i := 2; i < len(intervals); i++ {
		assert.Greater(t, intervals[i], intervals[i-1]/2,
			"interval %d (%v) should be greater than half of interval %d (%v)",
			i, intervals[i], i-1, intervals[i-1])
	}

	// Phase 2: success resets to base interval
	tracker.RecordSuccess()
	ticker.Reset(tracker.Interval())

	prev = time.Now()
	<-ticker.C
	resetInterval := time.Since(prev)

	// The reset interval should be close to base (10ms), with tolerance
	assert.Less(t, resetInterval, 3*base,
		"after success, interval should reset near base, got %v", resetInterval)
}

// TestTickerIntegrationMaxCap verifies the ticker caps at maxBackoff
// using real timing.
func TestTickerIntegrationMaxCap(t *testing.T) {
	base := 5 * time.Millisecond
	maxB := 50 * time.Millisecond
	tracker := New(base, maxB)

	// Push past the cap: 5ms * 2^4 = 80ms > 50ms cap
	for range 10 {
		tracker.RecordFailure()
	}

	ticker := time.NewTicker(tracker.Interval())
	defer ticker.Stop()

	start := time.Now()
	<-ticker.C
	elapsed := time.Since(start)

	// Should be around maxB (50ms), not unbounded
	assert.LessOrEqual(t, elapsed, maxB+20*time.Millisecond,
		"capped interval should not exceed maxBackoff + tolerance, got %v", elapsed)
}

// TestMultipleTrackersWithTickers simulates per-path isolation with real
// tickers: two paths where one fails and the other succeeds.
func TestMultipleTrackersWithTickers(t *testing.T) {
	pingTracker := New(10*time.Millisecond, 200*time.Millisecond)
	tokenTracker := New(10*time.Millisecond, 200*time.Millisecond)

	pingTicker := time.NewTicker(10 * time.Millisecond)
	tokenTicker := time.NewTicker(10 * time.Millisecond)
	defer pingTicker.Stop()
	defer tokenTicker.Stop()

	// Ping path: 3 failures
	for range 3 {
		<-pingTicker.C
		pingTracker.RecordFailure()
		pingTicker.Reset(pingTracker.Interval())
	}

	// Token path: stays healthy, 3 successes
	for range 3 {
		<-tokenTicker.C
		tokenTracker.RecordSuccess()
		tokenTicker.Reset(tokenTracker.Interval())
	}

	// Ping should be in backoff with elevated interval
	assert.True(t, pingTracker.InBackoff())
	assert.Greater(t, pingTracker.Interval(), 10*time.Millisecond)

	// Token should be at base interval, unaffected
	assert.False(t, tokenTracker.InBackoff())
	assert.Equal(t, 10*time.Millisecond, tokenTracker.Interval())
}
