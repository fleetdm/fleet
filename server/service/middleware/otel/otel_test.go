package otel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// The fast path registers the stdlib mux with the per-version expansions of a gorilla template but hands WithRouteTag the
// unexpanded template, so these two spellings of one route disagree. That mismatch is what these tests pin down.
const (
	stdlibPattern   = "/api/latest/fleet/hosts/{id}"
	gorillaTemplate = "/api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/{id:[0-9]+}"
)

// TestWithRouteTag serves through a stdlib ServeMux, which is what populates r.Pattern in production. The span column is
// the only one WithRouteTag controls.
func TestWithRouteTag(t *testing.T) {
	for _, tc := range []struct {
		name       string
		route      string // empty skips WithRouteTag, measuring what otelhttp does unaided
		wantSpan   string
		wantMetric string
	}{
		{"fast path template overrides the matched pattern", gorillaTemplate, gorillaTemplate, stdlibPattern},
		{"route equal to the pattern is redundant", stdlibPattern, stdlibPattern, stdlibPattern},
		{"without the helper otelhttp supplies the pattern", "", stdlibPattern, stdlibPattern},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spans := tracetest.NewSpanRecorder()
			tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
			reader := sdkmetric.NewManualReader()

			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
			if tc.route != "" {
				handler = WithRouteTag(tc.route, handler)
			}
			mux := http.NewServeMux()
			mux.Handle(stdlibPattern, otelhttp.NewHandler(handler, "test",
				otelhttp.WithTracerProvider(tracerProvider),
				otelhttp.WithMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))))
			mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com/api/latest/fleet/hosts/42", nil))
			require.NoError(t, tracerProvider.ForceFlush(t.Context()))

			ended := spans.Ended()
			require.Len(t, ended, 1)
			require.Equal(t, []string{tc.wantSpan}, routeAttrs(ended[0].Attributes()))
			require.Equal(t, []string{tc.wantMetric}, metricRouteAttrs(t, reader))
		})
	}
}

func routeAttrs(attrs []attribute.KeyValue) []string {
	var routes []string
	for _, attr := range attrs {
		if string(attr.Key) == "http.route" {
			routes = append(routes, attr.Value.AsString())
		}
	}
	return routes
}

func metricRouteAttrs(t *testing.T, reader *sdkmetric.ManualReader) []string {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &collected))
	for _, scope := range collected.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if histogram, ok := metric.Data.(metricdata.Histogram[float64]); ok && len(histogram.DataPoints) > 0 {
				return routeAttrs(histogram.DataPoints[0].Attributes.ToSlice())
			}
		}
	}
	return nil
}
