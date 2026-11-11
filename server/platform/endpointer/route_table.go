package endpointer

import (
	"net/http"
	"runtime"
	"slices"
	"strings"
	"sync"
	"weak"

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
	Prefix  bool
	Handler http.Handler
}

// Template returns the gorilla route template the route is registered with.
func (rt Route) Template() string {
	return strings.Replace(rt.Path, "/_version_/", "/{fleetversion:(?:"+strings.Join(rt.Versions, "|")+")}/", 1)
}

// RouteTable lists the routes registered through the endpointer on one router, in registration order.
type RouteTable struct {
	mu     sync.Mutex
	routes []Route
}

// Routes returns a copy of the recorded routes, in registration order.
func (t *RouteTable) Routes() []Route {
	t.mu.Lock()
	defer t.mu.Unlock()
	routes := make([]Route, 0, len(t.routes))
	for _, rt := range t.routes {
		routes = append(routes, rt.clone())
	}
	return routes
}

func (rt Route) clone() Route {
	rt.Methods = slices.Clone(rt.Methods)
	rt.Versions = slices.Clone(rt.Versions)
	return rt
}

func (t *RouteTable) add(rt Route) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.routes = append(t.routes, rt)
}

// routeTables maps each gorilla router to its RouteTable. The registration functions and the feature route functions only
// receive the *mux.Router, so this is how they reach the table without changing their signatures. The key is weak, and the
// entry is dropped when its router is collected, because tests build many routers. That only works while no recorded handler
// references its own router.
var routeTables sync.Map // weak.Pointer[mux.Router] -> *RouteTable

// RoutesOf returns the table of the routes registered through the endpointer on r.
func RoutesOf(r *mux.Router) *RouteTable {
	key := weak.Make(r)
	if t, ok := routeTables.Load(key); ok {
		return t.(*RouteTable)
	}
	t, loaded := routeTables.LoadOrStore(key, &RouteTable{})
	if !loaded {
		runtime.AddCleanup(r, func(key weak.Pointer[mux.Router]) { routeTables.Delete(key) }, key)
	}
	return t.(*RouteTable)
}

// handleRoute registers rt on r and records it in the router's RouteTable.
func handleRoute(r *mux.Router, rt Route) {
	var route *mux.Route
	if rt.Prefix {
		route = r.PathPrefix(rt.Template()).Handler(rt.Handler)
	} else {
		route = r.Handle(rt.Template(), rt.Handler)
	}
	rt = rt.clone()
	// gorilla keeps, and uppercases in place, the slice it is given.
	route.Name(rt.Name).Methods(slices.Clone(rt.Methods)...)
	RoutesOf(r).add(rt)
}
