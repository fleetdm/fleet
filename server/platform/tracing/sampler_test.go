package tracing

import (
	"encoding/binary"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// testRegistry returns a Registry pre-populated with the routes the sampler tests exercise. It lives next to its only consumer
// so registry_test.go can stay focused on Registry semantics.
func testRegistry() *Registry {
	r := NewRegistry()
	r.Register(http.MethodGet, "/healthz", TierNever)
	r.Register(http.MethodGet, "/version", TierNever)
	r.Register(http.MethodGet, "/metrics", TierNever)
	r.Register(http.MethodPost, "/api/osquery/distributed/read", TierHighVolume)
	r.Register(http.MethodPost, "/api/v1/osquery/distributed/read", TierHighVolume)
	r.Register(http.MethodPost, "/api/osquery/distributed/write", TierHighVolume)
	r.Register(http.MethodPost, "/api/fleet/orbit/config", TierHighVolume)
	r.Register(http.MethodHead, "/api/fleet/orbit/ping", TierHighVolume)
	r.Register(http.MethodHead, "/api/_version_/fleet/device/{token}/ping", TierHighVolume)
	r.Register(http.MethodGet, "/api/_version_/fleet/device/{token}/desktop", TierHighVolume)
	r.Register(http.MethodGet, "/api/_version_/fleet/hosts", TierStandard)
	r.Register(http.MethodGet, "/api/_version_/fleet/hosts/{id}", TierStandard)
	r.Register(http.MethodGet, "/api/_version_/fleet/queries", TierStandard)
	return r
}

// sample is a tiny helper that asks the sampler whether a span with the given name should be recorded. A pseudo random trace
// ID is used so the TraceIDRatioBased sampler's decision varies per call.
//
//nolint:gosec // test trace IDs, not security sensitive
func sample(t *testing.T, s *RouteTierSampler, name string) bool {
	t.Helper()
	var tid trace.TraceID
	binary.LittleEndian.PutUint64(tid[0:8], rand.Uint64())
	binary.LittleEndian.PutUint64(tid[8:16], rand.Uint64())
	res := s.ShouldSample(sdktrace.SamplingParameters{
		TraceID: tid,
		Name:    name,
		Kind:    trace.SpanKindServer,
	})
	return res.Decision == sdktrace.RecordAndSample
}

// sampleRate runs N trials and returns the observed sample rate.
func sampleRate(t *testing.T, s *RouteTierSampler, name string, n int) float64 {
	t.Helper()
	hits := 0
	for range n {
		if sample(t, s, name) {
			hits++
		}
	}
	return float64(hits) / float64(n)
}

// TestRouteTierSampler_NeverTierDropsUnconditionally locks in the invariant that TierNever paths are never sampled, both at
// default config and under the most aggressive override (force_full=true with ratios maxed). The force_full subtest is the
// stronger guarantee. The default config case is kept to make the absence of any default-time leak explicit.
func TestRouteTierSampler_NeverTierDropsUnconditionally(t *testing.T) {
	paths := []string{"GET /healthz", "GET /version", "GET /metrics"}

	cases := []struct {
		name  string
		apply func(*RouteTierSampler)
	}{
		{name: "default config", apply: func(*RouteTierSampler) {}},
		{name: "force_full with max ratios", apply: func(s *RouteTierSampler) { s.Apply(1.0, 1.0, true) }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewRouteTierSampler(testRegistry())
			c.apply(s)
			for _, p := range paths {
				for range 1000 {
					require.False(t, sample(t, s, p), "tierNever must drop %s", p)
				}
			}
		})
	}
}

// TestRouteTierSampler_AlwaysTierKeepsUnconditionally locks in the invariant that unclassified spans (cron, novel routes)
// always sample, both at default config and when ratios are forced to zero. The ratios=0 subtest is the stronger guarantee.
func TestRouteTierSampler_AlwaysTierKeepsUnconditionally(t *testing.T) {
	names := []string{
		"vuln.update_host_counts",                      // cron
		"POST /api/_version_/fleet/mdm/profiles/batch", // GitOps batch
		"POST /api/fleet/orbit/enroll",                 // enroll
		"some-future-endpoint-not-in-any-list",         // unknown
	}

	cases := []struct {
		name  string
		apply func(*RouteTierSampler)
	}{
		{name: "default config", apply: func(*RouteTierSampler) {}},
		{name: "ratios forced to zero", apply: func(s *RouteTierSampler) { s.Apply(0.0, 0.0, false) }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewRouteTierSampler(testRegistry())
			c.apply(s)
			for _, n := range names {
				for range 1000 {
					require.True(t, sample(t, s, n), "tierAlways must keep %s", n)
				}
			}
		})
	}
}

func TestRouteTierSampler_RatioSampling(t *testing.T) {
	const n = 100_000
	const tolerance = 0.005 // 0.5pp absolute, generous for 100k trials

	s := NewRouteTierSampler(testRegistry())
	s.Apply(0.1, 0.5, false) // visible ratios for stable comparisons

	highRate := sampleRate(t, s, "POST /api/osquery/distributed/read", n)
	require.InDelta(t, 0.1, highRate, tolerance, "high volume tier should track its configured ratio")

	stdRate := sampleRate(t, s, "GET /api/_version_/fleet/hosts", n)
	require.InDelta(t, 0.5, stdRate, tolerance, "standard tier should track its configured ratio")
}

func TestRouteTierSampler_ForceFull(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	s.Apply(0.0, 0.0, true) // ratios zero, force_full should override

	for _, name := range []string{
		"POST /api/osquery/distributed/read", // would be 0% via high volume
		"GET /api/_version_/fleet/hosts",     // would be 0% via standard
		"POST /api/fleet/orbit/enroll",       // already always
	} {
		t.Run(name, func(t *testing.T) {
			for range 1000 {
				require.True(t, sample(t, s, name),
					"force_full must override ratio based tiers")
			}
		})
	}
}

func TestRouteTierSampler_ApplyRaceFree(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())

	var (
		stop    atomic.Bool
		readers sync.WaitGroup
		writers sync.WaitGroup
	)

	// One writer flips ratios continuously.
	writers.Go(func() {
		for !stop.Load() {
			//nolint:gosec // test fuzz inputs, not security sensitive
			s.Apply(rand.Float64(), rand.Float64(), rand.IntN(2) == 0)
		}
	})

	// Several readers hammer ShouldSample.
	const readerCount = 8
	for range readerCount {
		readers.Go(func() {
			for !stop.Load() {
				_ = sample(t, s, "POST /api/osquery/distributed/read")
				_ = sample(t, s, "GET /healthz")
				_ = sample(t, s, "vuln.update_host_counts")
			}
		})
	}

	// Run for a tight window. The race detector will fail if there's a torn read.
	for range 50_000 {
		_ = sample(t, s, "POST /api/_version_/fleet/spec/teams")
	}
	stop.Store(true)
	readers.Wait()
	writers.Wait()
}

func TestRouteTierSampler_ClampOutOfRange(t *testing.T) {
	// Apply should clamp defensively even if a caller passes out of range ratios. The DB CHECK rejects these in practice.
	s := NewRouteTierSampler(testRegistry())
	s.Apply(-1.0, 5.0, false)
	st := s.state.Load()
	require.Equal(t, "TraceIDRatioBased{0}", st.highVolume.Description())
	require.Equal(t, "TraceIDRatioBased{1}", st.standard.Description())
}
