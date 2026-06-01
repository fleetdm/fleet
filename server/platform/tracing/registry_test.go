package tracing

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// testRegistry returns a Registry pre-populated with the routes used by the
// sampler tests. Sampler tests exercise the sampling math; the actual route
// tier policy lives outside the platform package (e.g.,
// server/service/tracing_tiers.go).
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

func TestRegistry_LookupNormalizesVersion(t *testing.T) {
	r := NewRegistry()
	r.Register(http.MethodGet, "/api/_version_/fleet/hosts", TierStandard)

	cases := []struct {
		name   string
		input  string
		want   Tier
		wantOK bool
	}{
		{
			name:   "regex version template",
			input:  "GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts",
			want:   TierStandard,
			wantOK: true,
		},
		{
			name:   "single-version regex template",
			input:  "GET /api/{fleetversion:(?:latest)}/fleet/hosts",
			want:   TierStandard,
			wantOK: true,
		},
		{
			name:   "already normalized form",
			input:  "GET /api/_version_/fleet/hosts",
			want:   TierStandard,
			wantOK: true,
		},
		{
			name:   "unregistered route returns TierAlways false",
			input:  "POST /not/in/registry",
			want:   TierAlways,
			wantOK: false,
		},
		{
			name:   "cron span name is not in registry",
			input:  "vuln.update_host_counts",
			want:   TierAlways,
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := r.Lookup(c.input)
			require.Equal(t, c.wantOK, ok)
			require.Equal(t, c.want, got)
		})
	}
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	r := NewRegistry()
	r.Register(http.MethodPost, "/foo", TierStandard)
	r.Register(http.MethodPost, "/foo", TierHighVolume) // overwrite

	got, ok := r.Lookup("POST /foo")
	require.True(t, ok)
	require.Equal(t, TierHighVolume, got)
}

func TestRegistry_ConcurrentReadersAndWriters(t *testing.T) {
	// Exercise the RWMutex under -race. Late-arriving registrations must
	// not corrupt concurrent lookups; this is what makes it safe for
	// bounded contexts to register at startup while the tracer provider
	// is already serving spans.
	r := NewRegistry()
	const writerCount = 4
	const readerCount = 8
	const iterations = 5000

	var wg sync.WaitGroup
	for w := range writerCount {
		wg.Go(func() {
			for i := range iterations {
				path := "/path/writer/" + itoa(w) + "/" + itoa(i)
				r.Register(http.MethodGet, path, TierStandard)
			}
		})
	}
	for range readerCount {
		wg.Go(func() {
			for range iterations {
				_, _ = r.Lookup("GET /path/writer/0/0")
			}
		})
	}
	wg.Wait()
}

// itoa is a tiny stdlib-free int-to-string for the concurrency test.
// Imported strconv would do the same; this avoids one import.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestNormalizeSpanName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"GET /healthz", "GET /healthz"},
		{
			"GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts",
			"GET /api/_version_/fleet/hosts",
		},
		{
			"POST /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/spec/teams",
			"POST /api/_version_/fleet/spec/teams",
		},
		{
			"GET /api/_version_/fleet/queries",
			"GET /api/_version_/fleet/queries",
		},
		{"vuln.update_host_counts", "vuln.update_host_counts"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			require.Equal(t, c.want, normalizeSpanName(c.in))
		})
	}
}
