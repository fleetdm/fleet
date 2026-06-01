package tracing

import (
	"encoding/binary"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// sample is a tiny helper that asks the sampler whether a span with the given
// name should be recorded. A pseudo-random trace ID is used so the
// TraceIDRatioBased sampler's decision varies per call.
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

func TestRouteTierSampler_NeverTier(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	for _, name := range []string{
		"GET /healthz",
		"GET /version",
		"GET /metrics",
	} {
		t.Run(name, func(t *testing.T) {
			for range 1000 {
				require.False(t, sample(t, s, name), "tierNever must drop every span")
			}
		})
	}
}

func TestRouteTierSampler_NeverIgnoresForceFull(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	s.Apply(1.0, 1.0, true) // force_full on, ratios maxed
	for range 1000 {
		require.False(t, sample(t, s, "GET /healthz"),
			"tierNever must drop even when force_full=true")
	}
}

func TestRouteTierSampler_AlwaysTierDefaults(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	// Unclassified names fall through to tierAlways (cron span, novel route).
	for _, name := range []string{
		"vuln.update_host_counts",                      // cron
		"POST /api/_version_/fleet/mdm/profiles/batch", // GitOps batch
		"POST /api/fleet/orbit/enroll",                 // enroll
		"some-future-endpoint-not-in-any-list",         // unknown
	} {
		t.Run(name, func(t *testing.T) {
			for range 1000 {
				require.True(t, sample(t, s, name), "tierAlways must keep every span")
			}
		})
	}
}

func TestRouteTierSampler_AlwaysTierIgnoresRatios(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	s.Apply(0.0, 0.0, false) // ratios zero, force_full off
	for range 1000 {
		require.True(t, sample(t, s, "vuln.update_host_counts"),
			"tierAlways must keep spans even when ratios are zero")
	}
}

func TestRouteTierSampler_RatioSampling(t *testing.T) {
	const n = 100_000
	const tolerance = 0.005 // 0.5pp absolute — generous for 100k trials

	s := NewRouteTierSampler(testRegistry())
	s.Apply(0.1, 0.5, false) // visible ratios for stable comparisons

	highRate := sampleRate(t, s, "POST /api/osquery/distributed/read", n)
	require.InDelta(t, 0.1, highRate, tolerance, "high-volume tier should track its configured ratio")

	stdRate := sampleRate(t, s, "GET /api/_version_/fleet/hosts", n)
	require.InDelta(t, 0.5, stdRate, tolerance, "standard tier should track its configured ratio")
}

func TestRouteTierSampler_VersionNormalization(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	s.Apply(0.0, 0.0, false) // make rates trivially observable: 0% on classified

	// Both versioned forms should normalize to the same tier (high-volume → 0%).
	versionedRegex := "HEAD /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/device/{token}/ping"
	for range 200 {
		require.False(t, sample(t, s, versionedRegex),
			"versioned device ping must be classified as high-volume after normalization")
	}

	// Standard tier with rate 0 — versioned host detail should never sample.
	versionedHostDetail := "GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/{id}"
	for range 200 {
		require.False(t, sample(t, s, versionedHostDetail),
			"versioned host detail must classify as standard after normalization")
	}
}

func TestRouteTierSampler_ForceFull(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	s.Apply(0.0, 0.0, true) // ratios zero — force_full should override

	for _, name := range []string{
		"POST /api/osquery/distributed/read", // would be 0% via high-volume
		"GET /api/_version_/fleet/hosts",     // would be 0% via standard
		"POST /api/fleet/orbit/enroll",       // already always
	} {
		t.Run(name, func(t *testing.T) {
			for range 1000 {
				require.True(t, sample(t, s, name),
					"force_full must override ratio-based tiers")
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
	// Apply should clamp defensively even if a caller passes out-of-range
	// ratios; the DB CHECK rejects these in practice.
	s := NewRouteTierSampler(testRegistry())
	s.Apply(-1.0, 5.0, false)
	st := s.state.Load()
	require.Equal(t, "TraceIDRatioBased{0}", st.highVolume.Description())
	require.Equal(t, "TraceIDRatioBased{1}", st.standard.Description())
}

func TestRouteTierSampler_Description(t *testing.T) {
	s := NewRouteTierSampler(testRegistry())
	require.Contains(t, s.Description(), "RouteTierSampler{")
	require.Contains(t, s.Description(), "forceFull=false")
}
