package tracing

import (
	"fmt"
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Default sampling ratios. These match the seeded trace_sampler_settings row so a freshly started server uses the same ratios
// as one that has polled the DB.
const (
	DefaultHighVolumeRatio = 0.001
	DefaultStandardRatio   = 0.02
)

// RouteTierSampler implements sdktrace.Sampler. The configured ratios live in an atomic.Pointer so SettingsPoller can swap them
// under a hot reader without locking. Tier classification is delegated to the Registry passed at construction time.
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

// NewRouteTierSampler returns a sampler initialized with the default ratios and force_full=false. The poller (if running)
// overwrites these on the first tick once it reads the DB row. The registry is consulted on every ShouldSample call. Routes
// added to the registry after construction are picked up immediately.
func NewRouteTierSampler(registry *Registry) *RouteTierSampler {
	s := &RouteTierSampler{registry: registry}
	s.state.Store(buildState(DefaultHighVolumeRatio, DefaultStandardRatio, false))
	return s
}

// Apply replaces the sampler's state atomically. Out of range ratios are clamped to [0, 1] as a defensive backstop. The DB
// CHECK constraints and the PATCH handler validation reject these earlier in the pipeline.
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

// ShouldSample implements sdktrace.Sampler. TierNever wins over ForceFull. Liveness probes should never trace, even during a
// 100% debug window. Unregistered spans (cron, MDM checkin, enroll, etc.) fall to TierAlways.
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

// Description implements sdktrace.Sampler. OTel uses it for diagnostic logging. The value should describe the sampler's
// behavior unambiguously.
func (s *RouteTierSampler) Description() string {
	st := s.state.Load()
	return fmt.Sprintf("RouteTierSampler{highVolume=%g,standard=%g,forceFull=%t}",
		samplerRatio(st.highVolume), samplerRatio(st.standard), st.forceFull)
}

// samplerRatio extracts the configured ratio from a TraceIDRatioBased sampler for description purposes only. The SDK does not
// expose the ratio directly, so we parse its description ("TraceIDRatioBased{0.001}").
func samplerRatio(s sdktrace.Sampler) float64 {
	var r float64
	if _, err := fmt.Sscanf(s.Description(), "TraceIDRatioBased{%f}", &r); err != nil {
		return -1
	}
	return r
}
