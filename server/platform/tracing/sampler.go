// Package tracing implements a route-aware head sampler for Fleet's OTEL
// trace export. The sampler classifies each span via a Registry — populated
// at startup by each bounded context owning its routes — and applies
// per-tier ratio sampling so the noisy hot-agent paths do not drown out
// the rare load-bearing paths (enroll, MDM command flows, cron jobs).
//
// Issue: https://github.com/fleetdm/fleet/issues/44652
//
// Runtime control lives in the trace_sampler_settings MySQL row. Each Fleet
// replica runs StartSettingsPoller which re-reads the row every 60 seconds
// and atomically swaps the sampler's state — no restart required to flip
// force_full during an incident debug window.
//
// Architecture: platform/tracing owns the mechanism (sampler + tier enum
// + registry). Each bounded context owns the policy (its own routes' tier
// classifications) and registers them at startup. This keeps the platform
// package free of cross-context coupling.
package tracing

import (
	"fmt"
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Default sampling ratios — match the seeded trace_sampler_settings row so
// a freshly started server uses the same ratios as one that has polled the
// DB.
const (
	DefaultHighVolumeRatio = 0.001
	DefaultStandardRatio   = 0.02
)

// RouteTierSampler implements sdktrace.Sampler. The configured ratios live
// in an atomic.Pointer so SettingsPoller can swap them under a hot reader
// without locking. Tier classification is delegated to the Registry passed
// at construction time.
type RouteTierSampler struct {
	state    atomic.Pointer[samplerState]
	registry *Registry
}

type samplerState struct {
	highVolume sdktrace.Sampler
	standard   sdktrace.Sampler
	always     sdktrace.Sampler
	never      sdktrace.Sampler
	forceFull  bool
}

// NewRouteTierSampler returns a sampler initialized with the default ratios
// and force_full=false. The poller (if running) overwrites these on the
// first tick once it reads the DB row. The registry is consulted on every
// ShouldSample call; routes added to the registry after construction are
// picked up immediately.
func NewRouteTierSampler(registry *Registry) *RouteTierSampler {
	s := &RouteTierSampler{registry: registry}
	s.state.Store(buildState(DefaultHighVolumeRatio, DefaultStandardRatio, false))
	return s
}

// Apply replaces the sampler's state atomically. Out-of-range ratios are
// clamped to [0, 1] as a defensive backstop; the DB CHECK constraints and
// the PATCH handler validation reject these earlier in the pipeline.
func (s *RouteTierSampler) Apply(highVolume, standard float64, forceFull bool) {
	s.state.Store(buildState(clamp01(highVolume), clamp01(standard), forceFull))
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}

func buildState(highVolume, standard float64, forceFull bool) *samplerState {
	return &samplerState{
		highVolume: sdktrace.TraceIDRatioBased(highVolume),
		standard:   sdktrace.TraceIDRatioBased(standard),
		always:     sdktrace.AlwaysSample(),
		never:      sdktrace.NeverSample(),
		forceFull:  forceFull,
	}
}

// ShouldSample implements sdktrace.Sampler. TierNever wins over ForceFull —
// liveness probes should never trace, even during a 100% debug window.
// Unregistered spans (cron, MDM checkin, enroll, etc.) fall to TierAlways.
func (s *RouteTierSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	st := s.state.Load()
	tier, _ := s.registry.Lookup(p.Name)
	if tier == TierNever {
		return st.never.ShouldSample(p)
	}
	if st.forceFull {
		return st.always.ShouldSample(p)
	}
	switch tier {
	case TierHighVolume:
		return st.highVolume.ShouldSample(p)
	case TierStandard:
		return st.standard.ShouldSample(p)
	case TierAlways:
		return st.always.ShouldSample(p)
	}
	return st.always.ShouldSample(p)
}

// Description implements sdktrace.Sampler. OTel uses it for diagnostic
// logging; the value should describe the sampler's behavior unambiguously.
func (s *RouteTierSampler) Description() string {
	st := s.state.Load()
	return fmt.Sprintf("RouteTierSampler{highVolume=%g,standard=%g,forceFull=%t}",
		samplerRatio(st.highVolume), samplerRatio(st.standard), st.forceFull)
}

// samplerRatio extracts the configured ratio from a TraceIDRatioBased
// sampler for description purposes only. The SDK does not expose the
// ratio directly, so we parse its description ("TraceIDRatioBased{0.001}").
func samplerRatio(s sdktrace.Sampler) float64 {
	var r float64
	if _, err := fmt.Sscanf(s.Description(), "TraceIDRatioBased{%f}", &r); err != nil {
		return -1
	}
	return r
}
