package endpointer

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gorilla/mux"
)

// Route is one route registered through the endpointer. It keeps what the caller registered, before the version placeholder is
// expanded into a gorilla regex, so the route list can be read without walking the gorilla router.
type Route struct {
	Name    string
	Methods []string
	// Path is the path as registered, with the /_version_/ placeholder and gorilla variable syntax, such as
	// "/api/_version_/fleet/hosts/{id:[0-9]+}".
	Path string
	// Versions are the API versions the route serves, "latest" included. They only apply to a Path with the placeholder.
	Versions []string
	// Prefix routes match the whole subtree under Path.
	Prefix bool
}

// Template returns the gorilla route template the route is registered with.
func (rt Route) Template() string {
	return strings.Replace(rt.Path, "/_version_/", "/{fleetversion:(?:"+strings.Join(rt.Versions, "|")+")}/", 1)
}

func (rt Route) clone() Route {
	rt.Methods = slices.Clone(rt.Methods)
	rt.Versions = slices.Clone(rt.Versions)
	return rt
}

// Router is Fleet's API router. It records every route registered through the endpointer, in registration order, and puts the
// route template on the request context of each request it serves. The routes are also registered on a gorilla router
// underneath, which still matches requests until the stdlib ServeMux replaces it. Register every route and call WrapHandlers
// before serving: none of them are safe to call concurrently with each other or with ServeHTTP.
type Router struct {
	gorilla *mux.Router
	routes  []*registeredRoute
}

type registeredRoute struct {
	Route
	// template is computed once at registration, since every request reads it.
	template string
	// handler is the route's current handler, without the route template. The router registers a fixed handler underneath
	// that calls this one, so WrapHandlers never has to change what the underlying router holds.
	handler http.Handler
}

func (rr *registeredRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rr.handler.ServeHTTP(w, req.WithContext(WithRouteTemplate(req.Context(), rr.template)))
}

// NewRouter returns an empty Router.
func NewRouter() *Router {
	return &Router{gorilla: mux.NewRouter()}
}

// ServeHTTP serves req with the route that matches it.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.gorilla.ServeHTTP(w, req)
}

// Gorilla returns the gorilla router underneath. TEMPORARY: it exists for the stdlib fast path and the router-wide middleware,
// and goes away with gorilla.
func (r *Router) Gorilla() *mux.Router { return r.gorilla }

// Routes returns a copy of the recorded routes, in registration order.
func (r *Router) Routes() []Route {
	routes := make([]Route, 0, len(r.routes))
	for _, rr := range r.routes {
		routes = append(routes, rr.clone())
	}
	return routes
}

// WrapHandlers replaces the handler of every route registered so far with wrap(route, handler). The route template stays on
// the request context of the wrapped handler.
func (r *Router) WrapHandlers(wrap func(rt Route, h http.Handler) http.Handler) {
	for _, rr := range r.routes {
		rr.handler = wrap(rr.clone(), rr.handler)
	}
}

// Handle registers handler for rt. The endpointer registers every API route through it; code outside the endpointer only
// needs it for a handler that is not a kit endpoint.
func (r *Router) Handle(rt Route, handler http.Handler) {
	rr := &registeredRoute{Route: rt.clone(), template: rt.Template(), handler: handler}
	var route *mux.Route
	if rt.Prefix {
		route = r.gorilla.PathPrefix(rr.template).Handler(rr)
	} else {
		route = r.gorilla.Handle(rr.template, rr)
	}
	route.Name(rt.Name)
	if len(rt.Methods) > 0 {
		// gorilla keeps, and uppercases in place, the slice it is given. An empty list would match no method at all.
		route.Methods(slices.Clone(rt.Methods)...)
	}
	if err := route.GetError(); err != nil {
		// Catches a developer mistake, such as a malformed regex in a route path, at startup. gorilla keeps a route it rejected
		// but never matches it, so the route would be listed without being served.
		panic(fmt.Sprintf("invalid route %s %s: %v", strings.Join(rt.Methods, ","), rt.Path, err))
	}
	r.routes = append(r.routes, rr)
}
