package otel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestWithRouteTag verifies our local replacement for otelhttp.WithRouteTag (removed upstream in v0.65.0) still annotates
// both the span and the metric labeler with http.route.
func TestWithRouteTag(t *testing.T) {
	for _, tc := range []struct {
		name  string
		route string
	}{
		{name: "static route", route: "/healthz"},
		{name: "templated route", route: "/api/v1/fleet/hosts/{id}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

			var labelerAttrs []attribute.KeyValue
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				labeler, _ := otelhttp.LabelerFromContext(r.Context())
				labelerAttrs = labeler.Get()
				w.WriteHeader(http.StatusOK)
			})

			handler := otelhttp.NewHandler(WithRouteTag(tc.route, inner), "test",
				otelhttp.WithTracerProvider(provider),
			)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://example.com/any", nil))
			require.Equal(t, http.StatusOK, rr.Code)

			require.NoError(t, provider.ForceFlush(t.Context()))
			spans := recorder.Ended()
			require.Len(t, spans, 1)

			want := attribute.String("http.route", tc.route)
			require.Contains(t, spans[0].Attributes(), want)
			require.Contains(t, labelerAttrs, want)
		})
	}
}
