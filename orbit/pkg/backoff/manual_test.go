package backoff

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManualBackoffAgainstHTTPServer spins up a local HTTP server that
// can be toggled between healthy (200) and unhealthy (401/500) to verify
// backoff behavior with real HTTP round-trips.
//
// This simulates the #44816 scenario: Desktop polls an endpoint that
// starts returning errors, and we verify the polling interval grows
// exponentially, then resets on recovery.
func TestManualBackoffAgainstHTTPServer(t *testing.T) {
	// --- Set up a test server that we can toggle between healthy and error ---
	var serverStatus int = http.StatusOK
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(serverStatus)
		fmt.Fprintf(w, `{"status":%d}`, serverStatus)
	}))
	defer srv.Close()

	client := srv.Client()

	base := 50 * time.Millisecond
	maxB := 500 * time.Millisecond
	tracker := New(base, maxB)

	ping := func() error {
		resp, err := client.Get(srv.URL + "/healthz")
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}

	// --- Phase 1: server healthy, 3 polls at base interval ---
	t.Log("=== Phase 1: Server healthy, polling at base interval ===")
	ticker := time.NewTicker(base)
	defer ticker.Stop()

	for i := range 3 {
		<-ticker.C
		err := ping()
		require.NoError(t, err, "healthy server should return 200")
		tracker.RecordSuccess()
		ticker.Reset(tracker.Interval())
		t.Logf("  Poll %d: OK, interval=%v, failures=%d", i+1, tracker.Interval(), tracker.ConsecutiveFailures())
	}
	assert.Equal(t, 0, tracker.ConsecutiveFailures())
	assert.False(t, tracker.InBackoff())

	// --- Phase 2: server returns 401 (expired token scenario) ---
	t.Log("=== Phase 2: Server returns 401 (simulating expired token) ===")
	serverStatus = http.StatusUnauthorized

	var intervals []time.Duration
	prev := time.Now()
	for i := range 5 {
		<-ticker.C
		elapsed := time.Since(prev)
		intervals = append(intervals, elapsed)
		prev = time.Now()

		err := ping()
		require.Error(t, err, "should get error from 401")
		tracker.RecordFailure()
		nextInterval := tracker.Interval()
		ticker.Reset(nextInterval)
		t.Logf("  Failure %d: waited %v, next=%v, failures=%d",
			i+1, elapsed.Round(time.Millisecond), nextInterval.Round(time.Millisecond), tracker.ConsecutiveFailures())
	}

	// Verify intervals grew (skip first which is from the previous base tick)
	for i := 2; i < len(intervals); i++ {
		assert.Greater(t, intervals[i], intervals[i-1]*7/10,
			"interval should generally grow: %v vs %v", intervals[i], intervals[i-1])
	}
	assert.True(t, tracker.InBackoff())
	assert.Equal(t, 5, tracker.ConsecutiveFailures())

	// --- Phase 3: server recovers, single success resets ---
	t.Log("=== Phase 3: Server recovers, backoff resets ===")
	serverStatus = http.StatusOK

	<-ticker.C
	err := ping()
	require.NoError(t, err)
	backoffDur := tracker.BackoffDuration()
	tracker.RecordSuccess()
	ticker.Reset(tracker.Interval())
	t.Logf("  Recovery: backoff lasted %v, interval reset to %v", backoffDur.Round(time.Millisecond), tracker.Interval())

	assert.False(t, tracker.InBackoff())
	assert.Equal(t, 0, tracker.ConsecutiveFailures())
	assert.Equal(t, base, tracker.Interval())

	// --- Phase 4: server returns 500 (server error scenario) ---
	t.Log("=== Phase 4: Server returns 500 (simulating server error) ===")
	serverStatus = http.StatusInternalServerError

	for i := range 3 {
		<-ticker.C
		err := ping()
		require.Error(t, err)
		tracker.RecordFailure()
		ticker.Reset(tracker.Interval())
		t.Logf("  500 error %d: next=%v, failures=%d",
			i+1, tracker.Interval().Round(time.Millisecond), tracker.ConsecutiveFailures())
	}
	assert.True(t, tracker.InBackoff())

	// --- Phase 5: server recovers again ---
	t.Log("=== Phase 5: Second recovery ===")
	serverStatus = http.StatusOK
	<-ticker.C
	err = ping()
	require.NoError(t, err)
	tracker.RecordSuccess()
	ticker.Reset(tracker.Interval())
	t.Logf("  Back to normal: interval=%v", tracker.Interval())
	assert.False(t, tracker.InBackoff())
	assert.Equal(t, base, tracker.Interval())
}

// TestManualBackoffServerDown simulates a complete server outage (connection
// refused) and verifies backoff applies the same as HTTP errors.
func TestManualBackoffServerDown(t *testing.T) {
	base := 50 * time.Millisecond
	maxB := 500 * time.Millisecond
	tracker := New(base, maxB)

	// Point at a port nothing listens on
	client := &http.Client{
		Timeout: 100 * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
		},
	}

	ticker := time.NewTicker(base)
	defer ticker.Stop()

	t.Log("=== Server down (connection refused), backoff should apply ===")
	for i := range 4 {
		<-ticker.C
		_, err := client.Get("https://127.0.0.1:19999/healthz")
		require.Error(t, err, "should fail to connect")
		tracker.RecordFailure()
		nextInterval := tracker.Interval()
		ticker.Reset(nextInterval)
		t.Logf("  Conn refused %d: next=%v, failures=%d",
			i+1, nextInterval.Round(time.Millisecond), tracker.ConsecutiveFailures())
	}

	assert.True(t, tracker.InBackoff())
	assert.Equal(t, 4, tracker.ConsecutiveFailures())
	// After 4 failures: 50ms * 2^4 = 800ms, capped to 500ms
	assert.LessOrEqual(t, tracker.Interval(), maxB)
}

// TestManualBackoffMaxCapWithRealServer verifies the backoff caps at
// maxBackoff with a real HTTP server returning continuous errors.
func TestManualBackoffMaxCapWithRealServer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	client := srv.Client()

	base := 20 * time.Millisecond
	maxB := 200 * time.Millisecond
	tracker := New(base, maxB)

	ticker := time.NewTicker(base)
	defer ticker.Stop()

	t.Log("=== Continuous 401s until cap is reached ===")
	var hitCap bool
	for i := range 10 {
		<-ticker.C
		resp, err := client.Get(srv.URL + "/device/expired-token/desktop")
		require.NoError(t, err) // HTTP succeeded, just got 401
		resp.Body.Close()

		tracker.RecordFailure()
		nextInterval := tracker.Interval()
		ticker.Reset(nextInterval)

		t.Logf("  Failure %d: next=%v, failures=%d",
			i+1, nextInterval.Round(time.Millisecond), tracker.ConsecutiveFailures())

		if nextInterval >= maxB {
			hitCap = true
		}
	}

	assert.True(t, hitCap, "should have hit the max backoff cap")
	assert.LessOrEqual(t, tracker.Interval(), maxB, "interval must not exceed max")
}
