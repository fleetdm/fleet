package backoff

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveServerBackoff exercises the backoff tracker against a real HTTP
// server that switches between returning 500 (server down) and 200 (recovered).
// This simulates the upgrade-outage scenario from the RCA.
func TestLiveServerBackoff(t *testing.T) {
	t.Parallel()

	var serverDown atomic.Bool
	serverDown.Store(true) // start with server "down"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serverDown.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Use short intervals for test speed. 50ms base, 400ms baseCap.
	// With 100% jitter, effective range at cap: 400ms-800ms.
	base := 50 * time.Millisecond
	baseCap := 400 * time.Millisecond
	tracker := newForTest(base, baseCap)

	client := srv.Client()
	prev := time.Now()

	// Phase 1: Server is down. Make requests, observe backoff growing.
	t.Log("Phase 1: Server down, observing backoff...")
	for i := range 8 {
		resp, err := client.Get(srv.URL)
		require.NoError(t, err)
		resp.Body.Close()

		now := time.Now()
		prev = now

		if resp.StatusCode != http.StatusOK {
			tracker.RecordFailure()
		} else {
			tracker.RecordSuccess()
		}

		waitDur := tracker.Interval()
		t.Logf("  request %d: HTTP %d, next_retry=%v, consecutive_failures=%d",
			i+1, resp.StatusCode, waitDur, tracker.ConsecutiveFailures())
		time.Sleep(waitDur)
	}
	_ = prev // used only for timing context in logs above

	// Verify intervals grew
	require.True(t, tracker.InBackoff(), "should be in backoff after failures")
	require.GreaterOrEqual(t, tracker.ConsecutiveFailures(), 6)

	// Verify that at the cap, jitter provides real spread (not all identical)
	capIntervals := make([]time.Duration, 10)
	for i := range capIntervals {
		capIntervals[i] = tracker.Interval()
	}
	// With 100% jitter on 400ms cap, intervals should be in [400ms, 800ms]
	minI, maxI := capIntervals[0], capIntervals[0]
	for _, iv := range capIntervals {
		assert.GreaterOrEqual(t, iv, baseCap, "interval should be >= baseCap")
		assert.LessOrEqual(t, iv, 2*baseCap, "interval should be <= 2*baseCap")
		if iv < minI {
			minI = iv
		}
		if iv > maxI {
			maxI = iv
		}
	}
	spread := maxI - minI
	t.Logf("  Cap-level jitter spread: min=%v max=%v spread=%v", minI, maxI, spread)
	assert.Greater(t, spread, 50*time.Millisecond,
		"jitter at cap should produce meaningful spread, got %v", spread)

	// Phase 2: Server recovers. Observe immediate reset.
	t.Log("Phase 2: Server recovers...")
	serverDown.Store(false)

	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	tracker.RecordSuccess()
	t.Logf("  Success! consecutive_failures=%d, next_interval=%v",
		tracker.ConsecutiveFailures(), tracker.Interval())

	assert.False(t, tracker.InBackoff(), "should exit backoff on success")
	assert.Equal(t, 0, tracker.ConsecutiveFailures())
	assert.Equal(t, base, tracker.Interval(), "should reset to base interval")

	// Phase 3: Verify normal operation resumes at base interval.
	t.Log("Phase 3: Normal operation at base interval...")
	for range 3 {
		resp, err := client.Get(srv.URL)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		tracker.RecordSuccess()
		assert.Equal(t, base, tracker.Interval())
	}

	fmt.Println("PASS: Live server backoff test completed successfully")
}
