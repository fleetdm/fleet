package tracing

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

type stubReader struct {
	mu       atomic.Pointer[fleet.TraceSamplerSettings]
	err      atomic.Pointer[error]
	getCalls atomic.Int32
}

func (s *stubReader) set(settings fleet.TraceSamplerSettings) {
	s.mu.Store(&settings)
}

func (s *stubReader) setErr(err error) {
	s.err.Store(&err)
}

func (s *stubReader) GetTraceSamplerSettings(ctx context.Context) (*fleet.TraceSamplerSettings, error) {
	s.getCalls.Add(1)
	if e := s.err.Load(); e != nil && *e != nil {
		return nil, *e
	}
	if cur := s.mu.Load(); cur != nil {
		out := *cur
		return &out, nil
	}
	return &fleet.TraceSamplerSettings{
		HighVolumeRatio: DefaultHighVolumeRatio,
		StandardRatio:   DefaultStandardRatio,
	}, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestStartSettingsPoller_AppliesInitialReadImmediately(t *testing.T) {
	r := &stubReader{}
	r.set(fleet.TraceSamplerSettings{
		HighVolumeRatio: 0.4,
		StandardRatio:   0.8,
		ForceFull:       true,
	})

	sampler := NewRouteTierSampler(NewRegistry())
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		StartSettingsPoller(ctx, sampler, r, discardLogger())
		close(done)
	}()

	// The first poll happens synchronously before the ticker arms, so once the
	// goroutine has run any amount it will have applied the settings. Wait until
	// the reader has been hit at least once, then cancel.
	require.Eventually(t, func() bool {
		return r.getCalls.Load() >= 1
	}, 1e9 /* 1s in nanoseconds — Eventually default tick */, 1e6)

	cancel()
	<-done

	// Sampler should reflect the applied settings.
	st := sampler.state.Load()
	require.True(t, st.forceFull)
}

func TestStartSettingsPoller_HandlesErrorGracefully(t *testing.T) {
	r := &stubReader{}
	r.setErr(errors.New("db unavailable"))

	sampler := NewRouteTierSampler(NewRegistry())
	// Capture current state — should remain unchanged after a failed poll.
	beforeForceFull := sampler.state.Load().forceFull

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		StartSettingsPoller(ctx, sampler, r, discardLogger())
		close(done)
	}()

	require.Eventually(t, func() bool {
		return r.getCalls.Load() >= 1
	}, 1e9, 1e6)

	cancel()
	<-done

	require.Equal(t, beforeForceFull, sampler.state.Load().forceFull,
		"sampler state must be unchanged when the read fails")
}
