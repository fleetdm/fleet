package otel

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHandler wraps an HTTP handler with OpenTelemetry instrumentation for a fixed route.
// It creates spans named as "{method} {route}" (e.g., "GET /healthz").
func WrapHandler(handler http.Handler, route string, config config.FleetConfig) http.Handler {
	if config.Logging.TracingEnabled && config.Logging.TracingType == "opentelemetry" {
		// Wrap with OTEL handler to create properly named spans: "{method} {route}"
		return otelhttp.NewHandler(
			otelhttp.WithRouteTag(route, handler),
			"", // Empty operation name - will be set by span name formatter
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return r.Method + " " + route
			}),
		)
	}
	return handler
}

// WrapHandlerDynamic wraps an HTTP handler with OpenTelemetry instrumentation using dynamic routes.
// It creates spans based on the actual request path (e.g., "GET /assets/app.js").
func WrapHandlerDynamic(handler http.Handler, config config.FleetConfig) http.Handler {
	if config.Logging.TracingEnabled && config.Logging.TracingType == "opentelemetry" {
		// Create a wrapper that instruments each request with its actual path
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use the actual request path as the route
			route := r.URL.Path
			instrumentedHandler := otelhttp.NewHandler(
				otelhttp.WithRouteTag(route, handler),
				"", // Empty operation name - will be set by span name formatter
				otelhttp.WithSpanNameFormatter(func(operation string, req *http.Request) string {
					return req.Method + " " + route
				}),
			)
			instrumentedHandler.ServeHTTP(w, r)
		})
	}
	return handler
}
