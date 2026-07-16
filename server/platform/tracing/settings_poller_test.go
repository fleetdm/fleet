package tracing

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

type stubReader struct {
	mu       atomic.Pointer[Settings]
	err      atomic.Pointer[error]
	getCalls atomic.Int32
}

func (s *stubReader) set(settings Settings) {
	s.mu.Store(&settings)
}

func (s *stubReader) setErr(err error) {
	s.err.Store(&err)
}

func (s *stubReader) GetTraceSamplerSettings(_ context.Context) (*Settings, error) {
	s.getCalls.Add(1)
	if e := s.err.Load(); e != nil && *e != nil {
		return nil, *e
	}
	if cur := s.mu.Load(); cur != nil {
		out := *cur
		return &out, nil
	}
	return &Settings{
		HighVolumeRatio: DefaultHighVolumeRatio,
		StandardRatio:   DefaultStandardRatio,
	}, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestStartSettingsPoller_AppliesInitialReadImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &stubReader{}
		r.set(Settings{
			HighVolumeRatio: 0.4,
			StandardRatio:   0.8,
			ForceFull:       true,
		})

		sampler := NewRouteTierSampler(NewRegistry())
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go StartSettingsPoller(ctx, sampler, r, discardLogger())

		// Wait blocks until every other goroutine in the bubble is durably blocked. The poller does its synchronous initial
		// read, applies, then blocks on the ticker. So once Wait returns, the apply has happened.
		synctest.Wait()

		require.Equal(t, int32(1), r.getCalls.Load(), "exactly one poll should have happened by now")
		st := sampler.state.Load()
		require.True(t, st.forceFull, "initial read must apply force_full")
	})
}

func TestStartSettingsPoller_HandlesErrorGracefully(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &stubReader{}
		r.setErr(errors.New("db unavailable"))

		sampler := NewRouteTierSampler(NewRegistry())
		// Capture current state. It should remain unchanged after a failed poll.
		beforeForceFull := sampler.state.Load().forceFull

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go StartSettingsPoller(ctx, sampler, r, discardLogger())

		synctest.Wait()

		require.Equal(t, int32(1), r.getCalls.Load(), "the failed poll still counts as one read")
		require.Equal(t, beforeForceFull, sampler.state.Load().forceFull,
			"sampler state must be unchanged when the read fails")
	})
}

func TestStartSettingsPoller_AppliesChangeOnTick(t *testing.T) {
	// Locks in the actual 60s ticker behavior. The old tests could only assert the initial synchronous read because waiting a
	// real minute per test was untenable. With synctest, advancing time is free.
	synctest.Test(t, func(t *testing.T) {
		r := &stubReader{}
		r.set(Settings{
			HighVolumeRatio: 0.4,
			StandardRatio:   0.8,
			ForceFull:       true,
		})

		sampler := NewRouteTierSampler(NewRegistry())
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go StartSettingsPoller(ctx, sampler, r, discardLogger())

		// Initial synchronous poll completes.
		synctest.Wait()
		require.Equal(t, int32(1), r.getCalls.Load())
		require.True(t, sampler.state.Load().forceFull)

		// Flip the stub to a new value. Advance past the next ticker fire; the synthetic clock advances while everything is
		// blocked, the ticker fires, the poller re-polls and applies the new state. Wait then ensures the poller has
		// re-blocked before we assert.
		r.set(Settings{
			HighVolumeRatio: 0.001,
			StandardRatio:   0.02,
			ForceFull:       false,
		})
		time.Sleep(settingsPollInterval + time.Nanosecond)
		synctest.Wait()

		require.Equal(t, int32(2), r.getCalls.Load(), "second poll must have fired after one ticker interval")
		require.False(t, sampler.state.Load().forceFull, "ticker poll must apply the new state")

		// One more tick with no underlying change should still call Get but should be a no-op for Apply. We verify by the
		// invariant that the state is unchanged from the previous assertion.
		time.Sleep(settingsPollInterval + time.Nanosecond)
		synctest.Wait()
		require.Equal(t, int32(3), r.getCalls.Load())
		require.False(t, sampler.state.Load().forceFull)
	})
}
