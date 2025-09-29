// internal/cbreaker/cbreaker.go
package cbreaker

import (
	"context"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/sony/gobreaker/v2"
)

// DB is a single shared breaker protecting your datastore.
var DB *gobreaker.CircuitBreaker[any]

// Init creates the global breaker with sane defaults and env overrides.
func Init(logger log.Logger) {
	st := gobreaker.Settings{
		Name:         "datastore",
		MaxRequests:  getUint32("CB_MAX_REQUESTS", 1, 1, math.MaxUint32), // Half-open trials (min 1)
		Interval:     getDur("CB_INTERVAL", 0),                           // Closed-state reset period
		BucketPeriod: getDur("CB_BUCKET_PERIOD", 0),                      // Rolling window buckets
		Timeout:      getDur("CB_OPEN_TIMEOUT", 10*time.Second),          // Open -> Half-open
		ReadyToTrip: func(c gobreaker.Counts) bool { // Trip rule
			thr := getUint32("CB_FAILURE_THRESHOLD", 5, 1, math.MaxUint32) // min 1
			return c.ConsecutiveFailures >= thr
		},
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

	if !getBool("CB_ENABLED", true) {
		st.ReadyToTrip = func(gobreaker.Counts) bool { return false }
		st.Timeout = 365 * 24 * time.Hour
		st.MaxRequests = math.MaxUint32
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

// getUint32 reads k, parses a base-10 unsigned integer with 32-bit size,
// clamps to [minVal, maxVal], and returns def if missing/invalid.
// minVal/maxVal let you enforce sensible bounds (e.g., min 1 for thresholds).
func getUint32(k string, def, minVal, maxVal uint32) uint32 {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	u64, err := strconv.ParseUint(v, 10, 32) // ensures <= MaxUint32
	if err != nil {
		return def
	}
	u := uint32(u64)
	if u < minVal {
		return minVal
	}
	if u > maxVal {
		return maxVal
	}
	return u
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
