package tracing

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_LookupNormalizes(t *testing.T) {
	// One fixture covers all input shapes that should resolve to the same registered route: fleetversion templates (multi
	// version and single version), mux regex constrained params, and already normalized inputs.
	r := NewRegistry()
	r.Register(http.MethodGet, "/api/_version_/fleet/hosts", TierStandard)
	r.Register(http.MethodGet, "/api/_version_/fleet/hosts/{id}", TierStandard)
	r.Register(http.MethodPatch, "/api/_version_/fleet/fleets/{fleet_id}/secrets", TierAlways)

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
			name:   "single version regex template",
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
			name:   "mux regex param {id:[0-9]+}",
			input:  "GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/{id:[0-9]+}",
			want:   TierStandard,
			wantOK: true,
		},
		{
			name:   "mux regex param {fleet_id:[0-9]+}",
			input:  "PATCH /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/fleets/{fleet_id:[0-9]+}/secrets",
			want:   TierAlways,
			wantOK: true,
		},
		{
			name:   "unregistered route",
			input:  "POST /not/in/registry",
			want:   TierAlways, // zero value. Sampler interprets the !ok as the catch all.
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

func TestRegistry_RegisterNormalizesSymmetrically(t *testing.T) {
	// Inverse of TestRegistry_LookupNormalizes: registering with the regex form must also resolve via lookup of the bare form.
	// This is the invariant the Register side normalization delivers.
	r := NewRegistry()
	r.Register(http.MethodGet, "/api/_version_/fleet/hosts/{id:[0-9]+}", TierStandard)

	got, ok := r.Lookup("GET /api/{fleetversion:(?:latest)}/fleet/hosts/{id}")
	require.True(t, ok, "regex form registration must be findable via simple form lookup")
	require.Equal(t, TierStandard, got)
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	t.Run("same form", func(t *testing.T) {
		r := NewRegistry()
		r.Register(http.MethodPost, "/foo", TierStandard)
		r.Register(http.MethodPost, "/foo", TierHighVolume)

		got, ok := r.Lookup("POST /foo")
		require.True(t, ok)
		require.Equal(t, TierHighVolume, got)
	})

	t.Run("different forms collide on the same normalized key", func(t *testing.T) {
		// {id} and {id:[0-9]+} are the same logical route after normalization. The second Register must overwrite the first
		// regardless of which surface form was used.
		r := NewRegistry()
		r.Register(http.MethodGet, "/api/_version_/fleet/hosts/{id}", TierStandard)
		r.Register(http.MethodGet, "/api/_version_/fleet/hosts/{id:[0-9]+}", TierHighVolume)

		got, _ := r.Lookup("GET /api/_version_/fleet/hosts/{id}")
		require.Equal(t, TierHighVolume, got)
	})
}

func TestRegistry_ConcurrentReadersAndWriters(t *testing.T) {
	// Exercise the RWMutex under -race. Late arriving registrations must not corrupt concurrent lookups. This is what makes it
	// safe for bounded contexts to register at startup while the tracer provider is already serving spans.
	r := NewRegistry()
	const writerCount = 4
	const readerCount = 8
	const iterations = 5000

	var wg sync.WaitGroup
	for w := range writerCount {
		wg.Go(func() {
			for i := range iterations {
				path := "/path/writer/" + strconv.Itoa(w) + "/" + strconv.Itoa(i)
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

func TestNormalizeSpanName(t *testing.T) {
	// Unit test for the helper. Lookup integration is covered separately. These cases pin the helper's behavior independent of
	// the map lookup.
	cases := []struct {
		in, want string
	}{
		{"GET /healthz", "GET /healthz"},
		{
			"GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts",
			"GET /api/_version_/fleet/hosts",
		},
		{
			"GET /api/_version_/fleet/queries",
			"GET /api/_version_/fleet/queries",
		},
		{"vuln.update_host_counts", "vuln.update_host_counts"},
		// Mux regex constraints on path params are stripped to the bare {name} form so the registry can stay decoupled from the
		// constraint syntax.
		{
			"GET /api/_version_/fleet/hosts/{id:[0-9]+}",
			"GET /api/_version_/fleet/hosts/{id}",
		},
		{
			"PATCH /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/fleets/{fleet_id:[0-9]+}/secrets",
			"PATCH /api/_version_/fleet/fleets/{fleet_id}/secrets",
		},
		// Params without a regex constraint pass through unchanged.
		{
			"GET /api/_version_/fleet/device/{token}/desktop",
			"GET /api/_version_/fleet/device/{token}/desktop",
		},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			require.Equal(t, c.want, normalizeSpanName(c.in))
		})
	}
}
