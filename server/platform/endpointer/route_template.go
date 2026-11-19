package endpointer

import "context"

// contextKeyRouteTemplate is the context key type for the route template.
type contextKeyRouteTemplate struct{}

var routeTemplateKey = contextKeyRouteTemplate{}

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
