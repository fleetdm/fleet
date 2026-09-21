package otel

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// WithRouteTag annotates the request's span and metrics with the http.route attribute. otelhttp dropped its own
// WithRouteTag in v0.65.0 because it now derives http.route from r.Pattern, which covers every call site whose route
// equals the stdlib pattern it is registered under. We will narrow this usage in #53612.
func WithRouteTag(route string, h http.Handler) http.Handler {
	attr := semconv.HTTPRoute(route)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace.SpanFromContext(r.Context()).SetAttributes(attr)

		labeler, _ := otelhttp.LabelerFromContext(r.Context())
		labeler.Add(attr)

		h.ServeHTTP(w, r)
	})
}

// WrapHandler wraps an HTTP handler with OpenTelemetry instrumentation for a fixed route.
// It creates spans named as "{method} {route}" (e.g., "GET /healthz").
func WrapHandler(handler http.Handler, route string, cfg config.FleetConfig) http.Handler {
	if cfg.OTELEnabled() {
		// Wrap with OTEL handler to create properly named spans: "{method} {route}"
		return otelhttp.NewHandler(
			WithRouteTag(route, handler),
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
func WrapHandlerDynamic(handler http.Handler, cfg config.FleetConfig) http.Handler {
	if cfg.OTELEnabled() {
		// Create a wrapper that instruments each request with its actual path
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use the actual request path as the route
			route := r.URL.Path
			instrumentedHandler := otelhttp.NewHandler(
				WithRouteTag(route, handler),
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
