package endpointer

import (
	"context"
	"net/http"
)

// contextKeyRouteTemplate is the context key type for the route template.
type contextKeyRouteTemplate struct{}

var routeTemplateKey = contextKeyRouteTemplate{}

// RouteTemplateRequestFunc copies the route template that the Router put on the request context into ctx, for callers whose
// ctx does not derive from the request's.
func RouteTemplateRequestFunc(ctx context.Context, r *http.Request) context.Context {
	tpl, ok := RouteTemplateFromContext(r.Context())
	if !ok {
		return ctx
	}
	return WithRouteTemplate(ctx, tpl)
}

// RouteTemplateFromContext returns the route template of the matched route, as the Router stores it. Returns "" and false if
// no template is in context.
func RouteTemplateFromContext(ctx context.Context) (string, bool) {
	tpl, ok := ctx.Value(routeTemplateKey).(string)
	return tpl, ok
}

// WithRouteTemplate returns a new context with the given route template value. Used by the Router, and by tests that need a
// template in context without running a router.
func WithRouteTemplate(ctx context.Context, tpl string) context.Context {
	return context.WithValue(ctx, routeTemplateKey, tpl)
}
