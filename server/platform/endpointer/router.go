package endpointer

import (
	"fmt"
	"maps"
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
	// handler is the route's current handler. The ServeMux holds a fixed handler that calls this one, since a ServeMux can't
	// replace a handler once registered.
	handler  http.Handler
	matchers []varMatcher
	// chain is the route middleware wrapped around a call to the current handler.
	chain http.Handler
	// patterns are the ServeMux patterns the route is registered as, one per method and version.
	patterns []muxPattern
}

func (rr *registeredRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if mw, ok := w.(*muxWriter); ok {
		mw.routed = true
		w = mw.ResponseWriter
	}
	if !rr.allows(req.PathValue) {
		rr.router.unmatched(w, req)
		return
	}
	rr.serve(w, req)
}

func (rr *registeredRoute) serve(w http.ResponseWriter, req *http.Request) {
	rr.chain.ServeHTTP(w, req.WithContext(WithRouteTemplate(req.Context(), rr.template)))
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

// muxPattern is one ServeMux pattern of a route, kept so the router can match a request against every pattern, not only the
// most specific one the ServeMux picks.
type muxPattern struct {
	pattern  string
	method   string   // "" matches every method
	segments []string // the path split on "/", a wildcard written "{name}"
	prefix   bool
	literals int
}

func newMuxPattern(pattern string, prefix bool) muxPattern {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		method, path = "", pattern
	}
	p := muxPattern{pattern: pattern, method: method}
	path = strings.TrimSuffix(path, "{$}")
	if strings.HasSuffix(path, "/") && prefix {
		p.prefix = true
		path = strings.TrimSuffix(path, "/")
	}
	p.segments = strings.Split(path, "/")
	for _, segment := range p.segments {
		if !strings.HasPrefix(segment, "{") {
			p.literals++
		}
	}
	return p
}

// match reports whether the decoded request path segments match the pattern, and returns the path values.
func (p muxPattern) match(segments []string) (map[string]string, bool) {
	if len(segments) < len(p.segments) || (!p.prefix && len(segments) != len(p.segments)) ||
		(p.prefix && len(segments) == len(p.segments)) {
		return nil, false
	}
	values := make(map[string]string)
	for i, segment := range p.segments {
		if name, isVar := strings.CutPrefix(segment, "{"); isVar {
			values[strings.TrimSuffix(name, "}")] = segments[i]
		} else if segment != segments[i] {
			return nil, false
		}
	}
	return values, true
}

// resolved is a route that takes a request, the pattern it matched, and the path values.
type resolved struct {
	rr      *registeredRoute
	pattern muxPattern
	values  map[string]string
}

// resolve returns the most specific route pattern serving method on the path whose constraints the values satisfy. Ties go
// to the route registered first.
func (r *Router) resolve(method string, segments []string) (resolved, bool) {
	var best resolved
	found := false
	for _, rr := range r.routes {
		for _, p := range rr.patterns {
			if p.method != "" && p.method != method && (method != http.MethodHead || p.method != http.MethodGet) {
				continue
			}
			values, ok := p.match(segments)
			if !ok || !rr.allows(func(name string) string { return values[name] }) {
				continue
			}
			if !found || p.literals > best.pattern.literals || (p.literals == best.pattern.literals && best.pattern.prefix && !p.prefix) {
				best, found = resolved{rr: rr, pattern: p, values: values}, true
			}
		}
	}
	return best, found
}

// allowMethods are the methods unmatched offers in an Allow header.
var allowMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
}

// unmatched answers a request the ServeMux matched without the route constraints: the route it picked had a path variable
// fail its constraint, or it found the path only on other methods. A ServeMux wildcard matches any segment, so the router
// enforces constraints itself and answers what a router matching on them would. It serves the request with a less specific
// route that takes it, or answers 405 when another method serves the path, and 404 otherwise.
func (r *Router) unmatched(w http.ResponseWriter, req *http.Request) {
	segments := strings.Split(req.URL.EscapedPath(), "/")
	for i, segment := range segments {
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		segments[i] = decoded
	}
	if match, ok := r.resolve(req.Method, segments); ok {
		routed := new(http.Request)
		*routed = *req
		routed.Pattern = match.pattern.pattern
		for name, value := range match.values {
			routed.SetPathValue(name, value)
		}
		match.rr.serve(w, routed)
		return
	}

	var allow []string
	for _, method := range allowMethods {
		if method == req.Method || (method == http.MethodGet && req.Method == http.MethodHead) || slices.Contains(allow, method) {
			continue
		}
		if match, ok := r.resolve(method, segments); ok {
			allow = append(allow, method)
			if match.pattern.method == http.MethodGet && method == http.MethodGet && req.Method != http.MethodHead {
				allow = append(allow, http.MethodHead) // the GET pattern serves HEAD
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

// muxWriter holds back the 405 the ServeMux writes when it finds the path only on other methods, since it does so without
// the route constraints. Until a route takes the request, it keeps the headers the ServeMux sets apart from the real ones.
type muxWriter struct {
	http.ResponseWriter
	header http.Header
	// routed is set by the route the ServeMux dispatched to, which then writes to the real ResponseWriter.
	routed bool
	// heldBack is set when the ServeMux answered 405.
	heldBack    bool
	wroteHeader bool
}

func (mw *muxWriter) Header() http.Header {
	if mw.header == nil {
		mw.header = make(http.Header)
	}
	return mw.header
}

func (mw *muxWriter) WriteHeader(code int) {
	if code == http.StatusMethodNotAllowed {
		mw.heldBack = true
		return
	}
	mw.wroteHeader = true
	maps.Copy(mw.ResponseWriter.Header(), mw.header)
	mw.ResponseWriter.WriteHeader(code)
}

func (mw *muxWriter) Write(b []byte) (int, error) {
	if mw.heldBack {
		return len(b), nil
	}
	if !mw.wroteHeader {
		mw.WriteHeader(http.StatusOK)
	}
	return mw.ResponseWriter.Write(b)
}

// NewRouter returns an empty Router.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// ServeHTTP serves req with the route that matches it.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	mw := &muxWriter{ResponseWriter: w}
	r.mux.ServeHTTP(mw, req)
	if mw.heldBack {
		r.unmatched(w, req)
	}
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
	rr.chain = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { rr.handler.ServeHTTP(w, req) })
	for _, mw := range slices.Backward(r.middleware) {
		rr.chain = mw(rt.clone(), rr.chain)
	}
	for _, pattern := range stdlibPatterns(rt) {
		register(r.mux, rt.Name, pattern, rr)
		rr.patterns = append(rr.patterns, newMuxPattern(pattern, rt.Prefix))
	}
	if rt.Prefix {
		// Without an exact pattern, the ServeMux redirects the path without its trailing slash to the subtree, and the
		// redirect would drop server.url_prefix, which is stripped before the request gets here.
		for _, pattern := range stdlibPatterns(Route{Methods: rt.Methods, Path: strings.TrimSuffix(rt.Path, "/"), Versions: rt.Versions}) {
			register(r.mux, rt.Name, pattern, http.NotFoundHandler())
		}
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
