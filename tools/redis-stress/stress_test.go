package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateWriteFlags(t *testing.T) {
	t.Parallel()
	const validRate = 1.0
	validDur := 10 * time.Minute
	validTTL := 10 * time.Minute

	cases := []struct {
		name     string
		workers  int
		rate     float64
		duration time.Duration
		keyTTL   time.Duration
		wantErr  string // substring; "" means expect success
	}{
		{"happy path", 1, validRate, validDur, validTTL, ""},
		{"workers=0", 0, validRate, validDur, validTTL, "workers must be >= 1"},
		{"workers=-1", -1, validRate, validDur, validTTL, "workers must be >= 1"},
		{"rate=0", 1, 0, validDur, validTTL, "rate must be > 0"},
		{"rate=-1", 1, -1, validDur, validTTL, "rate must be > 0"},
		{"rate=1e10 (period truncates to 0)", 1, 1e10, validDur, validTTL, "non-positive ticker period"},
		{"duration=0", 1, validRate, 0, validTTL, "duration must be > 0"},
		{"duration=-1s", 1, validRate, -1 * time.Second, validTTL, "duration must be > 0"},
		{"keyTTL=0", 1, validRate, validDur, 0, "key-ttl must be >= 1ms"},
		{"keyTTL=-1ms", 1, validRate, validDur, -time.Millisecond, "key-ttl must be >= 1ms"},
		{"keyTTL=500us (sub-ms)", 1, validRate, validDur, 500 * time.Microsecond, "key-ttl must be >= 1ms"},
		{"keyTTL=1ms (boundary)", 1, validRate, validDur, time.Millisecond, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			period, err := validateWriteFlags(c.workers, c.rate, c.duration, c.keyTTL)
			if c.wantErr == "" {
				require.NoError(t, err)
				require.Greater(t, period, time.Duration(0))
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestValidateRaceFlags(t *testing.T) {
	t.Parallel()
	validIters := 100
	validTTL := 4 * time.Minute

	cases := []struct {
		name       string
		workers    int
		iterations int
		ttl        time.Duration
		wantErr    string
	}{
		{"happy path", 1, validIters, validTTL, ""},
		{"workers=0", 0, validIters, validTTL, "workers must be >= 1"},
		{"workers=-1", -1, validIters, validTTL, "workers must be >= 1"},
		{"iterations=0", 1, 0, validTTL, "iterations must be >= 1"},
		{"iterations=-1", 1, -1, validTTL, "iterations must be >= 1"},
		{"ttl=0", 1, validIters, 0, "ttl must be >= 1ms"},
		{"ttl=-1ms", 1, validIters, -time.Millisecond, "ttl must be >= 1ms"},
		{"ttl=500us (sub-ms)", 1, validIters, 500 * time.Microsecond, "ttl must be >= 1ms"},
		{"ttl=1ms (boundary)", 1, validIters, time.Millisecond, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRaceFlags(c.workers, c.iterations, c.ttl)
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}
