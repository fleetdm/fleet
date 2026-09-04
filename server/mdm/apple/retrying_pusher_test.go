package apple_mdm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/nanopush"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/stretchr/testify/require"
)

// fakeInnerPusher returns canned per-id responses (default success) and a
// canned call-level error, recording every call.
type fakeInnerPusher struct {
	mu    sync.Mutex
	calls [][]string
	res   map[string]*nanomdm_push.Response
	err   error
	raw   bool // return res exactly as configured (possibly nil), no per-id fill
}

func (f *fakeInnerPusher) Push(_ context.Context, ids []string) (map[string]*nanomdm_push.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string(nil), ids...))
	if f.raw {
		return f.res, f.err
	}
	out := make(map[string]*nanomdm_push.Response, len(ids))
	for _, id := range ids {
		if r, ok := f.res[id]; ok {
			out[id] = r
		} else {
			out[id] = &nanomdm_push.Response{Id: "apns-" + id}
		}
	}
	return out, f.err
}

func (f *fakeInnerPusher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func transientErr() error {
	return fmt.Errorf("push HTTP status: 503: %w", &nanopush.JSONPushError{Reason: "InternalServerError"})
}

// newTestRetryingPusher returns a wrapper with no background loop and a
// controllable clock.
func newTestRetryingPusher(inner *fakeInnerPusher) (*RetryingPusher, *time.Time) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	p := newRetryingPusher(inner, slog.New(slog.DiscardHandler))
	p.clock = func() time.Time { return now }
	return p, &now
}

func requireWithinJitter(t *testing.T, base time.Time, d time.Duration, got time.Time) {
	t.Helper()
	require.False(t, got.Before(base.Add(time.Duration(float64(d)*0.8))), "before jitter window: %s", got)
	require.False(t, got.After(base.Add(time.Duration(float64(d)*1.2))), "after jitter window: %s", got)
}

func TestRetryingPusherPassthrough(t *testing.T) {
	ctx := t.Context()

	t.Run("results and error returned unchanged, even together", func(t *testing.T) {
		wantRes := map[string]*nanomdm_push.Response{
			"e1": {Id: "id1"},
			"e2": {Err: transientErr()},
		}
		wantErr := errors.New("partial failure")
		inner := &fakeInnerPusher{err: wantErr}
		p, _ := newTestRetryingPusher(inner)
		// bypass the canned-response builder to pin exact identity
		inner.res = wantRes
		res, err := p.Push(ctx, []string{"e1", "e2"})
		require.Equal(t, wantErr, err)
		require.Same(t, wantRes["e1"], res["e1"])
		require.Same(t, wantRes["e2"], res["e2"])
	})

	t.Run("nil map and nil error pass through with nothing scheduled", func(t *testing.T) {
		inner := &fakeInnerPusher{raw: true}
		p, _ := newTestRetryingPusher(inner)
		res, err := p.Push(ctx, []string{"e1"})
		require.NoError(t, err)
		require.Nil(t, res)
		require.Empty(t, p.pending)
	})
}

func TestRetryingPusherScheduling(t *testing.T) {
	ctx := t.Context()

	t.Run("transient failure schedules the first backoff", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, now := newTestRetryingPusher(inner)
		_, err := p.Push(ctx, []string{"e1"})
		require.NoError(t, err)
		require.Len(t, p.pending, 1)
		require.Equal(t, 1, p.pending["e1"].attempt)
		requireWithinJitter(t, *now, 30*time.Second, p.pending["e1"].nextAttempt)
	})

	t.Run("call-level failure with no per-device verdict schedules every id", func(t *testing.T) {
		// a push-cert or push-info retrieval failure leaves the affected ids
		// out of the response map entirely; they must still be retried —
		// this is the common single-host lock/wipe shape.
		inner := &fakeInnerPusher{raw: true, res: map[string]*nanomdm_push.Response{}, err: errors.New("retrieving push cert for topic: timeout")}
		p, now := newTestRetryingPusher(inner)
		_, err := p.Push(ctx, []string{"e1", "e2"})
		require.Error(t, err)
		require.Len(t, p.pending, 2)
		requireWithinJitter(t, *now, 30*time.Second, p.pending["e1"].nextAttempt)
	})

	t.Run("per-device transport error is transient too", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: errors.New("dial tcp: connection refused")}}}
		p, _ := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		require.Len(t, p.pending, 1)
	})

	t.Run("permanent rejections are never scheduled", func(t *testing.T) {
		for _, reason := range []string{
			APNSReasonUnregistered, APNSReasonExpiredToken, APNSReasonBadDeviceToken,
			APNSReasonDeviceTokenNotForTopic, APNSReasonExpiredProviderToken, APNSReasonForbidden,
		} {
			t.Run(reason, func(t *testing.T) {
				inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{
					"e1": {Err: fmt.Errorf("push HTTP status: 410: %w", &nanopush.JSONPushError{Reason: reason})},
				}}
				p, _ := newTestRetryingPusher(inner)
				_, _ = p.Push(ctx, []string{"e1"})
				require.Empty(t, p.pending)
			})
		}
	})

	t.Run("a fresh failure restarts a backed-off entry's ladder", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, now := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		firstAttemptAt := p.pending["e1"].nextAttempt

		// at the first step, a duplicate failure collapses without rescheduling
		_, _ = p.Push(ctx, []string{"e1"})
		require.Equal(t, 1, p.pending["e1"].attempt)
		require.Equal(t, firstAttemptAt, p.pending["e1"].nextAttempt)

		// after a failed background retry escalates the entry, a fresh
		// enqueue-time failure restarts the ladder
		*now = now.Add(time.Minute)
		p.flushDue(ctx)
		require.Equal(t, 2, p.pending["e1"].attempt)
		_, _ = p.Push(ctx, []string{"e1"})
		require.Equal(t, 1, p.pending["e1"].attempt)
		requireWithinJitter(t, *now, 30*time.Second, p.pending["e1"].nextAttempt)
	})

	t.Run("enrollments without push info are never scheduled", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: nanomdm_pushsvc.ErrIdNotFound}}}
		p, _ := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		require.Empty(t, p.pending)
	})

	t.Run("an observed success clears a pending retry", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, _ := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		require.Len(t, p.pending, 1)
		inner.res = nil // next push succeeds
		_, _ = p.Push(ctx, []string{"e1"})
		require.Empty(t, p.pending)
	})

	t.Run("overflow drops the oldest entry", func(t *testing.T) {
		inner := &fakeInnerPusher{}
		p, now := newTestRetryingPusher(inner)
		var buf strings.Builder
		p.logger = slog.New(slog.NewTextHandler(&buf, nil))
		for i := range retryPendingCap {
			p.scheduleRetryLocked(ctx, fmt.Sprintf("e%05d", i), *now)
			*now = now.Add(time.Millisecond)
		}
		require.Len(t, p.pending, retryPendingCap)
		p.scheduleRetryLocked(ctx, "one-more", *now)
		require.Len(t, p.pending, retryPendingCap)
		require.NotContains(t, p.pending, "e00000", "oldest entry dropped")
		require.Contains(t, p.pending, "one-more")
		require.Contains(t, buf.String(), "level=WARN")
	})
}

func TestRetryingPusherFlush(t *testing.T) {
	ctx := t.Context()

	t.Run("escalates through the backoffs then gives up", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, now := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		require.Equal(t, 1, inner.callCount())

		// advance past each backoff (max + jitter margin) and flush; the
		// entry escalates 2m -> 8m -> 30m and is dropped after the last.
		for i, wantNext := range []time.Duration{2 * time.Minute, 8 * time.Minute, 30 * time.Minute} {
			*now = p.pending["e1"].nextAttempt.Add(time.Second)
			p.flushDue(ctx)
			require.Equal(t, i+2, p.pending["e1"].attempt)
			requireWithinJitter(t, *now, wantNext, p.pending["e1"].nextAttempt)
		}
		*now = p.pending["e1"].nextAttempt.Add(time.Second)
		p.flushDue(ctx)
		require.Empty(t, p.pending, "dropped after the last backoff")
		require.Equal(t, 5, inner.callCount(), "initial push + four retries")

		// nothing left to do
		p.flushDue(ctx)
		require.Equal(t, 5, inner.callCount())
	})

	t.Run("success on retry removes the entry", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, now := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		inner.res = nil // retry succeeds
		*now = now.Add(time.Minute)
		p.flushDue(ctx)
		require.Empty(t, p.pending)
	})

	t.Run("entries not yet due are left alone", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, _ := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		p.flushDue(ctx) // now == schedule time, backoff not elapsed
		require.Equal(t, 1, inner.callCount())
	})

	t.Run("a retry that finds the enrollment gone drops the entry", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, now := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		inner.res = map[string]*nanomdm_push.Response{"e1": {Err: nanomdm_pushsvc.ErrIdNotFound}}
		*now = now.Add(time.Minute)
		p.flushDue(ctx)
		require.Empty(t, p.pending)
	})

	t.Run("call-level failure keeps entries due without escalating", func(t *testing.T) {
		inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
		p, now := newTestRetryingPusher(inner)
		_, _ = p.Push(ctx, []string{"e1"})
		inner.raw = true
		inner.res = nil
		inner.err = errors.New("cert store down")
		*now = now.Add(time.Minute)
		p.flushDue(ctx)
		require.Len(t, p.pending, 1)
		require.Equal(t, 1, p.pending["e1"].attempt, "not the device's fault")
	})
}

func TestRetryingPusherBreaker(t *testing.T) {
	ctx := t.Context()

	inner := &fakeInnerPusher{res: map[string]*nanomdm_push.Response{"e1": {Err: transientErr()}}}
	p, now := newTestRetryingPusher(inner)
	_, _ = p.Push(ctx, []string{"e1"})

	// consecutive call-level failures open the breaker
	inner.raw = true
	inner.err = errors.New("apns is down")
	for range retryBreakerThreshold {
		_, err := p.Push(ctx, []string{"eX"})
		require.Error(t, err)
	}
	require.True(t, p.breakerOpenUntil.After(*now), "breaker open")

	// the retry loop is paused while open
	*now = now.Add(time.Minute)
	before := inner.callCount()
	p.flushDue(ctx)
	require.Equal(t, before, inner.callCount(), "no retry flush while the breaker is open")

	// fresh enqueue-time pushes still pass through
	_, _ = p.Push(ctx, []string{"eY"})
	require.Equal(t, before+1, inner.callCount())

	// after the cooldown the loop resumes, and a success closes the streak
	inner.raw = false
	inner.err = nil
	inner.res = nil
	*now = p.breakerOpenUntil.Add(time.Second)
	p.flushDue(ctx)
	require.Empty(t, p.pending, "retry delivered after cooldown")
	require.Zero(t, p.consecutiveFailures)
}

func TestRetryingPusherStop(t *testing.T) {
	inner := &fakeInnerPusher{}
	p := NewRetryingPusher(inner, slog.New(slog.DiscardHandler))
	_, err := p.Push(t.Context(), []string{"e1"})
	require.NoError(t, err)
	p.Stop() // must return promptly and not leak the loop goroutine (-race)
}
