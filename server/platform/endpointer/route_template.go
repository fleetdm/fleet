package endpointer

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

// contextKeyRouteTemplate is the context key type for the mux route template.
type contextKeyRouteTemplate struct{}

var routeTemplateKey = contextKeyRouteTemplate{}

// RouteTemplateRequestFunc captures the gorilla/mux route template for the
// matched request and stores it in the context.
func RouteTemplateRequestFunc(ctx context.Context, r *http.Request) context.Context {
	route := mux.CurrentRoute(r)
	if route == nil {
		return ctx
	}
	tpl, err := route.GetPathTemplate()
	if err != nil {
		// Only happens when a route has no path, which Fleet never registers.
		return ctx
	}
	return context.WithValue(ctx, routeTemplateKey, tpl)
}

// RouteTemplateFromContext returns the mux route template stored by
// RouteTemplateRequestFunc. Returns "" and false if no template is in context.
func RouteTemplateFromContext(ctx context.Context) (string, bool) {
	tpl, ok := ctx.Value(routeTemplateKey).(string)
	return tpl, ok
}

// WithRouteTemplate returns a new context with the given route template value.
// Intended for tests that need to simulate what RouteTemplateRequestFunc would
// have stored without running a real mux router.
func WithRouteTemplate(ctx context.Context, tpl string) context.Context {
	return context.WithValue(ctx, routeTemplateKey, tpl)
}
