package service

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestFastPathSpanNamesMatchGorillaTemplates pins the span-naming contract across the routing change.
//
// The fast path cannot use otelmux, which names spans from the matched gorilla route, so it wraps each promoted handler with
// otelhttp and supplies the route template itself. Both routers therefore have to produce "METHOD <gorilla template>", because
// server/service/tracing_tiers.go classifies routes for trace sampling by span name: a promoted route whose span name drifted
// would silently miss its registry entry and fall back to TierAlways, sampling a high-volume agent endpoint at 100%.
func TestFastPathSpanNamesMatchGorillaTemplates(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	cfg := config.TestConfig()
	cfg.Logging.TracingEnabled = true
	cfg.Logging.TracingType = "opentelemetry"
	require.True(t, cfg.OTELEnabled(), "the fast path is only installed on the OTEL tracing path")

	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h := MakeHandler(svc, cfg, slog.New(slog.DiscardHandler), limitStore, nil, nil, productionFeatureRoutes(t, svc))
	router := h.(interface{ Router() *mux.Router }).Router()

	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		route.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
		return nil
	}))
	handler := newFastPathHandler(router, nil, cfg, slog.New(slog.DiscardHandler))
	require.IsType(t, &fastPathHandler{}, handler)

	cases := []struct {
		name         string
		method, path string
		wantSpan     string
	}{
		{
			name:   "literal agent route on the fast path",
			method: "POST", path: "/api/osquery/distributed/write",
			wantSpan: "POST /api/osquery/distributed/write",
		},
		{
			name:   "versioned route on the fast path keeps the unexpanded template",
			method: "GET", path: "/api/latest/fleet/config",
			wantSpan: "GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/config",
		},
		{
			name:   "route with a constrained variable on the fast path",
			method: "GET", path: "/api/latest/fleet/hosts/1",
			wantSpan: "GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/{id:[0-9]+}",
		},
		{
			name:   "excluded route falls back to gorilla and is named by otelmux",
			method: "GET", path: "/api/latest/fleet/hosts/identifier/somehost",
			wantSpan: "GET /api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/identifier/{identifier}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recorder.Reset()
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(c.method, c.path, nil))
			require.Equal(t, http.StatusOK, rr.Code)

			var names []string
			for _, span := range recorder.Ended() {
				names = append(names, span.Name())
			}
			require.Containsf(t, names, c.wantSpan,
				"span name must stay %q so the trace sampler tier registry still matches it; got %v", c.wantSpan, names)
		})
	}
}

// TestFastPathSpanNamesNormalizeToTracingTiers is the other half of the contract: the sampler normalizes a span name before
// looking it up, so the names produced above have to resolve to the tiers server/service/tracing_tiers.go registers.
func TestFastPathSpanNamesNormalizeToTracingTiers(t *testing.T) {
	registry := tracing.NewRegistry()
	RegisterTracingTiers(registry)

	for _, c := range []struct{ span, why string }{
		{"POST /api/osquery/distributed/write", "highest-volume agent endpoint"},
		{"POST /api/fleet/orbit/config", "orbit config poll"},
	} {
		t.Run(c.span, func(t *testing.T) {
			_, found := registry.Lookup(c.span)
			require.Truef(t, found, "%s (%s) must resolve in the tier registry, otherwise it samples at 100%%", c.span, c.why)
		})
	}
}
