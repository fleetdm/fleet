package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/require"
)

// writeGzFile writes payload gzipped to dir and returns the path.
func writeGzFile(t *testing.T, dir string, payload []byte) string {
	t.Helper()

	path := filepath.Join(dir, "nvdcve-1.1-2026.json.gz")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(payload)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))

	return path
}

func TestGunzipFileToDisk(t *testing.T) {
	const limit int64 = 1024

	for _, tc := range []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{name: "under limit", size: limit - 1},
		{name: "at limit", size: limit},
		{name: "over limit", size: limit + 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			payload := bytes.Repeat([]byte("a"), int(tc.size))
			gzPath := writeGzFile(t, dir, payload)
			outPath := filepath.Join(dir, "nvdcve-1.1-2026.json")

			err := gunzipFileToDisk(gzPath, dir, limit)
			if tc.wantErr {
				require.Error(t, err)
				// A truncated feed left on disk parses as valid JSON to a later
				// stage, so it must not survive the failure.
				require.NoFileExists(t, outPath)
				return
			}

			require.NoError(t, err)
			got, err := os.ReadFile(outPath)
			require.NoError(t, err)
			require.Equal(t, payload, got)
		})
	}
}

func newRateLimitError(reset time.Time) error {
	rlErr := &github.RateLimitError{
		Rate: github.Rate{Reset: github.Timestamp{Time: reset}},
		// Error() dereferences Response.Request.
		Response: &http.Response{Request: &http.Request{Method: http.MethodGet, URL: &url.URL{}}},
	}
	// Wrapped the same way downloadLatestRelease wraps it.
	return fmt.Errorf("download cve feed: %w", fmt.Errorf("get cve asset path: %w", rlErr))
}

func newAbuseRateLimitError(retryAfter *time.Duration) error {
	abuseErr := &github.AbuseRateLimitError{
		RetryAfter: retryAfter,
		// Error() dereferences Response.Request.
		Response: &http.Response{Request: &http.Request{Method: http.MethodGet, URL: &url.URL{}}},
	}
	return fmt.Errorf("download cve feed: %w", fmt.Errorf("get cve asset path: %w", abuseErr))
}

func TestSeedFromLatestRelease(t *testing.T) {
	// Each case runs in a synctest bubble, whose fake clock starts at this time
	// and advances instantly whenever the code under test waits.
	now := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	logger := slog.New(slog.DiscardHandler)

	// run returns the number of download attempts, the wait before each retry
	// (measured between attempts) and the returned error.
	run := func(t *testing.T, results ...error) (int, []time.Duration, error) {
		t.Helper()
		var (
			calls int
			waits []time.Duration
			err   error
		)
		synctest.Test(t, func(t *testing.T) {
			require.True(t, now.Equal(time.Now()), "unexpected synctest start time %s", time.Now())
			var lastAttempt time.Time
			download := func() error {
				require.Less(t, calls, len(results), "unexpected extra download attempt")
				if calls > 0 {
					waits = append(waits, time.Since(lastAttempt))
				}
				lastAttempt = time.Now()
				calls++
				return results[calls-1]
			}
			err = seedFromLatestRelease(t.Context(), logger, download)

			var total time.Duration
			for _, w := range waits {
				total += w
			}
			require.Equal(t, total, time.Since(now), "waited after the last attempt")
		})
		return calls, waits, err
	}

	t.Run("succeeds first try", func(t *testing.T) {
		calls, waits, err := run(t, nil)
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Empty(t, waits)
	})

	t.Run("generic errors retry at fixed interval then fail", func(t *testing.T) {
		boom := errors.New("boom")
		calls, waits, err := run(t, boom, boom, boom)
		require.ErrorIs(t, err, boom)
		require.Equal(t, seedMaxAttempts, calls)
		require.Equal(t, []time.Duration{seedRetryInterval, seedRetryInterval}, waits)
	})

	t.Run("rate limited waits until reset", func(t *testing.T) {
		calls, waits, err := run(t, newRateLimitError(now.Add(8*time.Minute+23*time.Second)), nil)
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, []time.Duration{8*time.Minute + 23*time.Second + rateLimitResetBuffer}, waits)
	})

	t.Run("rate limit already reset waits only the buffer", func(t *testing.T) {
		calls, waits, err := run(t, newRateLimitError(now.Add(-time.Minute)), nil)
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, []time.Duration{rateLimitResetBuffer}, waits)
	})

	t.Run("rate limit reset beyond cap fails without waiting", func(t *testing.T) {
		calls, waits, err := run(t, newRateLimitError(now.Add(maxRateLimitWait+time.Minute)), nil)
		var rlErr *github.RateLimitError
		require.ErrorAs(t, err, &rlErr)
		require.Equal(t, 1, calls)
		require.Empty(t, waits)
	})

	t.Run("rate limit reset exactly at cap still waits", func(t *testing.T) {
		calls, waits, err := run(t, newRateLimitError(now.Add(maxRateLimitWait)), nil)
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, []time.Duration{maxRateLimitWait + rateLimitResetBuffer}, waits)
	})

	t.Run("secondary rate limit waits for retry-after", func(t *testing.T) {
		retryAfter := 90 * time.Second
		calls, waits, err := run(t, newAbuseRateLimitError(&retryAfter), nil)
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, []time.Duration{retryAfter + rateLimitResetBuffer}, waits)
	})

	t.Run("secondary rate limit without retry-after uses fixed interval", func(t *testing.T) {
		calls, waits, err := run(t, newAbuseRateLimitError(nil), nil)
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, []time.Duration{seedRetryInterval}, waits)
	})

	t.Run("rate limited on last attempt fails", func(t *testing.T) {
		boom := errors.New("boom")
		calls, waits, err := run(t, boom, boom, newRateLimitError(now.Add(time.Minute)))
		var rlErr *github.RateLimitError
		require.ErrorAs(t, err, &rlErr)
		require.Equal(t, seedMaxAttempts, calls)
		require.Equal(t, []time.Duration{seedRetryInterval, seedRetryInterval}, waits)
	})

	t.Run("deadline during a wait stops retrying", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			var calls int
			download := func() error {
				calls++
				return newRateLimitError(time.Now().Add(8 * time.Minute))
			}
			err := seedFromLatestRelease(ctx, logger, download)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Equal(t, 1, calls)
			require.Equal(t, time.Minute, time.Since(now))
		})
	})
}
