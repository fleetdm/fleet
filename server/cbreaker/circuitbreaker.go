// internal/cbreaker/cbreaker.go
package cbreaker

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/sony/gobreaker/v2"
)

// DB is a single shared breaker protecting your datastore.
// We use T=any because different calls return different types.
var DB *gobreaker.CircuitBreaker[any]

// Init creates the global breaker with sane defaults and env overrides.
func Init(logger log.Logger) {
	st := gobreaker.Settings{
		Name:         "datastore",
		MaxRequests:  uint32(getInt("CB_MAX_REQUESTS", 1)),      // Half-open trials (0 => 1)
		Interval:     getDur("CB_INTERVAL", 0),                  // Closed-state reset period
		BucketPeriod: getDur("CB_BUCKET_PERIOD", 0),             // Rolling window buckets (0 => fixed window)
		Timeout:      getDur("CB_OPEN_TIMEOUT", 10*time.Second), // Open -> Half-open
		ReadyToTrip: func(c gobreaker.Counts) bool { // Trip rule
			// Default is >5 consecutive failures; keep it explicit and env-tunable
			return c.ConsecutiveFailures >= uint32(getInt("CB_FAILURE_THRESHOLD", 5))
		},
		// Don't count caller cancellations/timeouts as "failures" for the breaker.
		IsSuccessful: func(err error) bool {
			return err == nil || err == context.Canceled || err == context.DeadlineExceeded
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			level.Warn(logger).Log(
				"msg", "circuit breaker state change",
				"name", name, "from", from.String(), "to", to.String(),
			)
		},
	}

	// Enable/disable via CB_ENABLED
	if !getBool("CB_ENABLED", true) {
		// A disabled breaker that never trips (still constructed so callers can use it).
		st.ReadyToTrip = func(gobreaker.Counts) bool { return false }
		st.Timeout = 365 * 24 * time.Hour
		st.MaxRequests = ^uint32(0)
		st.Name += " (disabled)"
	}

	DB = gobreaker.NewCircuitBreaker[any](st)
}

// --- tiny env helpers (no external deps) ---

func getBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getInt(k string, def int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getDur(k string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
