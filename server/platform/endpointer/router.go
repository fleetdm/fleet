package endpointer

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Route is one route registered through the endpointer. It keeps what the caller registered, before the version placeholder is
// expanded, so the route list can be read without introspecting the ServeMux.
type Route struct {
	Name    string
	Methods []string
	// Path is the path as registered, with the /_version_/ placeholder and the variable syntax the route table has always used,
	// such as "/api/_version_/fleet/hosts/{id:[0-9]+}". A variable can carry one of the constraints in varConstraints.
	Path string
	// Versions are the API versions the route serves, "latest" included. They only apply to a Path with the placeholder.
	Versions []string
	// Prefix routes match the whole subtree under Path.
	Prefix bool
}

// Template returns the route template: Path with the version placeholder spelled as the list of versions. API-only
// authorization, span names and request body size overrides are keyed on it, so its format must not change.
func (rt Route) Template() string {
	return strings.Replace(rt.Path, "/_version_/", "/{fleetversion:(?:"+strings.Join(rt.Versions, "|")+")}/", 1)
}

func (rt Route) clone() Route {
	rt.Methods = slices.Clone(rt.Methods)
	rt.Versions = slices.Clone(rt.Versions)
	return rt
}

// Router is Fleet's API router, a stdlib ServeMux underneath. It records every route registered through the endpointer, in
// registration order, and puts the route template on the request context of each request it serves. Call Use before
// registering routes, and register every route and call WrapHandlers before serving.
type Router struct {
	mux        *http.ServeMux
	routes     []*registeredRoute
	middleware []func(rt Route, next http.Handler) http.Handler
}

type registeredRoute struct {
	Route
	router *Router
	// template is computed once at registration, since every request reads it.
	template string
	matchers []varMatcher
	// served is the route's middleware around a call to handler.
	served http.Handler
	// handler is the route's current handler. The ServeMux holds a fixed handler that calls this one, since a ServeMux can't
	// replace a handler once registered.
	handler http.Handler
}

func (rr *registeredRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !rr.allows(req.PathValue) {
		rr.router.constraintFailed(w, req)
		return
	}
	rr.served.ServeHTTP(w, req.WithContext(WithRouteTemplate(req.Context(), rr.template)))
}

// allows reports whether every constrained path variable, as pathValue returns it, satisfies its constraint.
func (rr *registeredRoute) allows(pathValue func(name string) string) bool {
	for _, m := range rr.matchers {
		if !m.allow(pathValue(m.name)) {
			return false
		}
	}
	return true
}

// allowMethods are the methods constraintFailed tries.
var allowMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
}

// constraintFailed answers a request whose route had a path variable fail its constraint. A ServeMux wildcard matches any
// segment, so the router enforces constraints itself, and answers what a router that matched on them would: 405 when another
// method serves the path, and 404 otherwise, rather than letting the request reach a decoder.
func (r *Router) constraintFailed(w http.ResponseWriter, req *http.Request) {
	var allow []string
	for _, method := range allowMethods {
		if method == req.Method || (method == http.MethodGet && req.Method == http.MethodHead) {
			continue
		}
		probe := *req
		probe.Method = method
		h, pattern := r.mux.Handler(&probe)
		other, ok := h.(*registeredRoute)
		if !ok || pattern == "" {
			continue
		}
		values := patternValues(pattern, req.URL.EscapedPath())
		if other.allows(func(name string) string { return values[name] }) {
			allow = append(allow, method)
			if method == http.MethodGet {
				allow = append(allow, http.MethodHead)
			}
		}
	}
	if len(allow) == 0 {
		http.NotFound(w, req)
		return
	}
	slices.Sort(allow)
	w.Header().Set("Allow", strings.Join(allow, ", "))
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

// patternValues returns the path variables of a ServeMux pattern, as "METHOD /path", that matched escapedPath. A ServeMux
// only returns them for the request it serves.
func patternValues(pattern, escapedPath string) map[string]string {
	_, patternPath, _ := strings.Cut(pattern, " ")
	patternSegments := strings.Split(patternPath, "/")
	pathSegments := strings.Split(escapedPath, "/")
	values := make(map[string]string)
	for i, segment := range patternSegments {
		if i >= len(pathSegments) || !strings.HasPrefix(segment, "{") || segment == "{$}" {
			continue
		}
		if v, err := url.PathUnescape(pathSegments[i]); err == nil {
			values[strings.Trim(segment, "{}")] = v
		}
	}
	return values
}

// NewRouter returns an empty Router.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// ServeHTTP serves req with the route that matches it.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Routes returns a copy of the recorded routes, in registration order.
func (r *Router) Routes() []Route {
	routes := make([]Route, 0, len(r.routes))
	for _, rr := range r.routes {
		routes = append(routes, rr.clone())
	}
	return routes
}

// Use adds route middleware: every route registered after it runs mw(route, next) around its handler, the first middleware
// added outermost. It only runs for a request a route matched, never for a 404 or 405 from the router itself.
func (r *Router) Use(mw func(rt Route, next http.Handler) http.Handler) {
	if len(r.routes) > 0 {
		panic("endpointer.Router: Use must be called before any route is registered")
	}
	r.middleware = append(r.middleware, mw)
}

// WrapHandlers replaces the handler of every route registered so far with wrap(route, handler), inside the route middleware.
// The route template stays on the request context of the wrapped handler.
func (r *Router) WrapHandlers(wrap func(rt Route, h http.Handler) http.Handler) {
	for _, rr := range r.routes {
		rr.handler = wrap(rr.clone(), rr.handler)
	}
}

// Handle registers handler for rt. The endpointer registers every API route through it; code outside the endpointer only
// needs it for a handler that is not a kit endpoint. It panics on a route the ServeMux can't tell apart from one already
// registered, or on a variable constraint it doesn't support, since either is a mistake in the fixed route table.
func (r *Router) Handle(rt Route, handler http.Handler) {
	rr := &registeredRoute{Route: rt.clone(), router: r, template: rt.Template(), handler: handler}
	matchers, err := varMatchers(rt.Path)
	if err != nil {
		panic(fmt.Sprintf("route %s: %v", rt.Name, err))
	}
	rr.matchers = matchers
	rr.served = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { rr.handler.ServeHTTP(w, req) })
	for _, mw := range slices.Backward(r.middleware) {
		rr.served = mw(rt.clone(), rr.served)
	}
	for _, pattern := range stdlibPatterns(rt) {
		register(r.mux, rt.Name, pattern, rr)
	}
	r.routes = append(r.routes, rr)
}

// register adds pattern to the ServeMux, turning the panic net/http raises for a conflicting pattern into one that names the
// route and the fix.
func register(m *http.ServeMux, name, pattern string, h http.Handler) {
	defer func() {
		if p := recover(); p != nil {
			if err, ok := p.(error); ok && strings.Contains(err.Error(), "conflicts with pattern") {
				panic(fmt.Sprintf("route %s: %q is ambiguous on a stdlib ServeMux; change the route's path so it no longer conflicts: %v",
					name, pattern, err))
			}
			panic(p)
		}
	}()
	m.Handle(pattern, h)
}

var varPattern = regexp.MustCompile(`{([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?}`)

// stdlibPatterns returns the ServeMux patterns for rt, one per method and version. Each version is a literal segment, so a
// request naming an unknown version is a 404 at the router, and variable constraints are dropped, because varMatchers
// enforces them.
func stdlibPatterns(rt Route) []string {
	path := varPattern.ReplaceAllString(rt.Path, "{$1}")
	switch {
	case rt.Prefix && !strings.HasSuffix(path, "/"):
		path += "/"
	case !rt.Prefix && strings.HasSuffix(path, "/"):
		path += "{$}" // a ServeMux pattern ending in a slash matches the whole subtree otherwise
	}
	paths := []string{path}
	if strings.Contains(path, "/_version_/") {
		versions := slices.Compact(slices.Sorted(slices.Values(rt.Versions)))
		paths = paths[:0]
		for _, version := range versions {
			paths = append(paths, strings.Replace(path, "/_version_/", "/"+version+"/", 1))
		}
	}
	methods := rt.Methods
	if slices.Contains(methods, http.MethodGet) {
		// A GET pattern also serves HEAD, and a HEAD pattern next to the GET pattern of a sibling path is ambiguous.
		methods = slices.DeleteFunc(slices.Clone(methods), func(method string) bool { return method == http.MethodHead })
	}
	if len(methods) == 0 {
		return paths
	}
	patterns := make([]string, 0, len(methods)*len(paths))
	for _, method := range methods {
		for _, path := range paths {
			patterns = append(patterns, strings.ToUpper(method)+" "+path)
		}
	}
	return patterns
}

// varMatcher validates a path variable against the constraint in its route path, without running a regex.
type varMatcher struct {
	name  string
	allow func(string) bool
}

// varConstraints are the variable constraints a route path can use, each with the character set that enforces it.
var varConstraints = map[string]string{
	"[0-9]+":        "0123456789",
	"[a-f0-9-]+":    "0123456789abcdef-",
	"[a-zA-Z0-9-]+": "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-",
}

func varMatchers(path string) ([]varMatcher, error) {
	var matchers []varMatcher
	for _, m := range varPattern.FindAllStringSubmatch(path, -1) {
		name, constraint := m[1], m[2]
		if constraint == "" {
			continue
		}
		charset, ok := varConstraints[constraint]
		if !ok {
			return nil, fmt.Errorf("unsupported constraint %q on path variable %q", constraint, name)
		}
		matchers = append(matchers, varMatcher{name: name, allow: charsetMatcher(charset)})
	}
	return matchers, nil
}

func charsetMatcher(set string) func(string) bool {
	var allowed [256]bool
	for i := 0; i < len(set); i++ {
		allowed[set[i]] = true
	}
	return func(s string) bool {
		if s == "" {
			return false
		}
		for i := 0; i < len(s); i++ {
			if !allowed[s[i]] {
				return false
			}
		}
		return true
	}
}
