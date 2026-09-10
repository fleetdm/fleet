package service

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

// newRouterForTest builds the real API route table and replaces every handler with one that reports the route it belongs to,
// the mux vars it resolved, and the route template it found in context. Comparing those strings across the two routers catches
// a mis-dispatch, a lost path variable, and a missing route template alike.
func newRouterForTest(t *testing.T) *mux.Router {
	t.Helper()
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	provider, ok := h.(interface{ Router() *mux.Router })
	require.True(t, ok, "MakeHandler should return the fast-path handler by default")
	router := provider.Router()

	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		name := route.GetName()
		route.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// RouteTemplateRequestFunc is what the real transport runs; it reads the gorilla route when there is one and
			// leaves the context alone otherwise, which is how the fast path supplies the template.
			tpl, _ := endpointer.RouteTemplateFromContext(endpointer.RouteTemplateRequestFunc(r.Context(), r))
			_, _ = w.Write([]byte(name + " tpl=" + tpl + " vars=" + formatVars(mux.Vars(r))))
		})
		return nil
	}))
	return router
}

func formatVars(vars map[string]string) string {
	parts := make([]string, 0, len(vars))
	for k, v := range vars {
		parts = append(parts, k+"="+v)
	}
	slices.Sort(parts)
	return "{" + strings.Join(parts, ",") + "}"
}

var (
	numericVarRe = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*:\[0-9\]\+\}`)
	charsetVarRe = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*:\[[a-zA-Z0-9-]*\]\+\}`)
	plainVarRe   = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)
)

// sampleRequestPath builds a concrete path that the given route template must match.
func sampleRequestPath(tpl string) string {
	p := tpl
	if m := fleetVersionVar.FindStringSubmatch(p); m != nil {
		p = fleetVersionVar.ReplaceAllString(p, strings.Split(m[1], "|")[0])
	}
	p = numericVarRe.ReplaceAllString(p, "1")
	p = charsetVarRe.ReplaceAllString(p, "abc123")
	p = plainVarRe.ReplaceAllString(p, "abc")
	if strings.HasSuffix(p, "/") {
		p += "sub"
	}
	return p
}

func responseFor(h http.Handler, method, path string) (int, string, string) {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(method, path, nil))
	return rr.Code, rr.Body.String(), rr.Header().Get("Location")
}

// TestFastPathMatchesGorillaForEveryRoute sends one request per registered route through both routers and requires an
// identical response. Any difference means the fast path claimed a request gorilla would have routed elsewhere, dropped a path
// variable, or lost the route template the API-only endpoint restriction depends on.
func TestFastPathMatchesGorillaForEveryRoute(t *testing.T) {
	router := newRouterForTest(t)

	type sample struct{ method, path, route string }
	var samples []sample
	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := route.GetPathTemplate()
		if err != nil {
			return nil //nolint:nilerr // a route with no path is not reachable
		}
		methods, err := route.GetMethods()
		if err != nil || len(methods) == 0 {
			return nil //nolint:nilerr
		}
		for _, method := range methods {
			samples = append(samples, sample{method: method, path: sampleRequestPath(tpl), route: route.GetName()})
		}
		return nil
	}))
	require.Greater(t, len(samples), 400, "expected the full route table to be walked")

	handler := newFastPathHandler(router, nil, config.TestConfig(), slog.New(slog.DiscardHandler))
	require.IsType(t, &fastPathHandler{}, handler, "no route should have been rejected by the stdlib mux")

	for _, s := range samples {
		wantCode, wantBody, _ := responseFor(router, s.method, s.path)
		gotCode, gotBody, _ := responseFor(handler, s.method, s.path)
		require.Equalf(t, wantCode, gotCode, "status differs for %s %s (route %s)", s.method, s.path, s.route)
		require.Equalf(t, wantBody, gotBody, "dispatch differs for %s %s (route %s)", s.method, s.path, s.route)
	}
	t.Logf("compared %d method+path samples", len(samples))
}

// TestFastPathMatchesGorillaForRejectedRequests covers the requests a stdlib pattern matches more loosely than the gorilla
// template it replaced, or matches the same but answers differently. All of them have to reach gorilla.
func TestFastPathMatchesGorillaForRejectedRequests(t *testing.T) {
	router := newRouterForTest(t)
	handler := newFastPathHandler(router, nil, config.TestConfig(), slog.New(slog.DiscardHandler))

	cases := []struct {
		name         string
		method, path string
	}{
		{"unknown path", "GET", "/api/latest/fleet/nope/nothing/here"},
		{"method mismatch", "DELETE", "/api/osquery/distributed/write"},
		{"HEAD on a GET route is a 405", "HEAD", "/api/latest/fleet/hosts/1"},
		{"non-numeric id fails the route constraint", "GET", "/api/latest/fleet/hosts/not-a-number"},
		{"unknown API version", "GET", "/api/v9/fleet/hosts/1"},
		{"identifier beats the numeric host id", "GET", "/api/latest/fleet/hosts/identifier/device_mapping"},
		{"host by identifier", "GET", "/api/latest/fleet/hosts/identifier/somehost"},
		{"host subresource excluded alongside it", "GET", "/api/latest/fleet/hosts/1/device_mapping"},
		{"batch summary beats batch host results", "GET", "/api/latest/fleet/scripts/batch/summary/host_results"},
		{"both spellings of profile resend", "POST", "/api/latest/fleet/hosts/1/configuration_profiles/resend/resend"},
		{"encoded separator in a path variable", "GET", "/api/latest/fleet/device/abc%2Fdef"},
		{"encoded space in a path variable", "GET", "/api/latest/fleet/device/abc%20def"},
		{"encoded percent in a path variable", "GET", "/api/latest/fleet/device/abc%25def"},
		{"empty path segment redirects", "GET", "/api/latest//fleet/config"},
		{"dot segment redirects", "GET", "/api/latest/fleet/./config"},
		{"dot dot segment redirects", "GET", "/api/latest/fleet/hosts/../config"},
		{"trailing slash on an exact route", "GET", "/api/latest/fleet/config/"},
		{"query string is ignored by matching", "GET", "/api/latest/fleet/config?page=1"},
		{"websocket live results prefix route", "GET", "/api/latest/fleet/results/anything"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantCode, wantBody, wantLocation := responseFor(router, c.method, c.path)
			gotCode, gotBody, gotLocation := responseFor(handler, c.method, c.path)
			require.Equal(t, wantCode, gotCode)
			require.Equal(t, wantBody, gotBody)
			require.Equal(t, wantLocation, gotLocation)
		})
	}
}

// TestFastPathServesHotAgentRoutes proves the agent endpoints are actually served by the stdlib mux rather than quietly
// falling through to gorilla, which would make the equivalence tests above pass while changing nothing. A request routed by
// gorilla has a current route attached; one routed by the fast path does not.
func TestFastPathServesHotAgentRoutes(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	router := h.(interface{ Router() *mux.Router }).Router()

	var servedByGorilla bool
	var routeTemplate string
	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		route.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			servedByGorilla = mux.CurrentRoute(r) != nil
			routeTemplate, _ = endpointer.RouteTemplateFromContext(r.Context())
		})
		return nil
	}))
	handler := newFastPathHandler(router, nil, config.TestConfig(), slog.New(slog.DiscardHandler))

	for _, c := range []struct{ method, path, wantTemplate string }{
		{"POST", "/api/osquery/distributed/write", "/api/osquery/distributed/write"},
		{"POST", "/api/osquery/config", "/api/osquery/config"},
		{"POST", "/api/osquery/log", "/api/osquery/log"},
		{"POST", "/api/fleet/orbit/config", "/api/fleet/orbit/config"},
		{"HEAD", "/api/fleet/orbit/ping", "/api/fleet/orbit/ping"},
		{"HEAD", "/api/fleet/device/ping", "/api/fleet/device/ping"},
		{"POST", "/api/mdm/microsoft/management", "/api/mdm/microsoft/management"},
		{
			"HEAD", "/api/latest/fleet/device/6f36ab2c-1a40-4c3f-9f8e-1b3c2d4e5f60/ping",
			"/api/{fleetversion:(?:v1|2022-04|latest)}/fleet/device/{token}/ping",
		},
		{"GET", "/api/latest/fleet/hosts/1", "/api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/{id:[0-9]+}"},
	} {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			servedByGorilla, routeTemplate = false, ""
			code, _, _ := responseFor(handler, c.method, c.path)
			require.Equal(t, http.StatusOK, code)
			require.False(t, servedByGorilla, "route should be served by the stdlib fast path")
			require.Equal(t, c.wantTemplate, routeTemplate, "fast path must still supply the route template")
		})
	}
}

// TestFastPathAppliesRouterMiddlewareInGorillaOrder checks the router middleware that gorilla applies on a match is applied
// again, in the same order, to a route the fast path serves. Getting this wrong would silently drop gzip, client IP
// extraction, or HTTP signature verification for most requests.
func TestFastPathAppliesRouterMiddlewareInGorillaOrder(t *testing.T) {
	tag := func(name string) mux.MiddlewareFunc {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Middleware", name)
				next.ServeHTTP(w, r)
			})
		}
	}
	middlewares := []mux.MiddlewareFunc{tag("first"), tag("second"), tag("third")}

	router := mux.NewRouter()
	for _, mw := range middlewares {
		router.Use(mw)
	}
	router.Handle("/api/v1/fleet/config", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).Methods("GET").Name("get_config")

	handler := newFastPathHandler(router, middlewares, config.TestConfig(), slog.New(slog.DiscardHandler))
	require.IsType(t, &fastPathHandler{}, handler)

	viaGorilla := httptest.NewRecorder()
	router.ServeHTTP(viaGorilla, httptest.NewRequest("GET", "/api/v1/fleet/config", nil))
	viaFastPath := httptest.NewRecorder()
	handler.ServeHTTP(viaFastPath, httptest.NewRequest("GET", "/api/v1/fleet/config", nil))

	require.Equal(t, []string{"first", "second", "third"}, viaGorilla.Header().Values("X-Middleware"))
	require.Equal(t, viaGorilla.Header().Values("X-Middleware"), viaFastPath.Header().Values("X-Middleware"))
}

// TestFastPathExclusionsCoverEveryAmbiguousRoute fails when a route is added that a stdlib ServeMux cannot tell apart from an
// existing one. Without an entry in fastPathExcluded for both halves the fast path would silently turn itself off, so this
// keeps the list in sync with the route table.
func TestFastPathExclusionsCoverEveryAmbiguousRoute(t *testing.T) {
	router := newRouterForTest(t)

	registered := make(map[string]struct{})
	fast := http.NewServeMux()
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := route.GetPathTemplate()
		if err != nil || strings.HasSuffix(tpl, "/") {
			return nil //nolint:nilerr
		}
		methods, err := route.GetMethods()
		if err != nil || len(methods) != 1 {
			return nil //nolint:nilerr
		}
		if _, ok := varMatchers(tpl); !ok {
			return nil
		}
		key := unversionedKey(methods[0], tpl)
		registered[key] = struct{}{}
		for _, pattern := range stdlibPatterns(tpl) {
			if conflictErr := handleNoConflict(fast, methods[0]+" "+pattern, noop); conflictErr != nil {
				require.Containsf(t, fastPathExcluded, key,
					"route %q is ambiguous on a stdlib ServeMux; add it and the route it collides with to fastPathExcluded (%v)",
					key, conflictErr)
			}
		}
		return nil
	}))

	for key := range fastPathExcluded {
		require.Containsf(t, registered, key,
			"fastPathExcluded entry %q no longer matches a registered route and should be removed", key)
	}
}

// TestFastPathHandlerIsIntrospectableByEndpointValidation covers the startup path in cmd/fleet, which validates the endpoint
// catalog against the handler MakeHandler returns and panics if it cannot read the route table out of it.
func TestFastPathHandlerIsIntrospectableByEndpointValidation(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	router := h.(interface{ Router() *mux.Router }).Router()

	wrapped := apiendpoints.Validate(h)
	direct := apiendpoints.Validate(router)
	if direct == nil {
		require.NoError(t, wrapped)
		return
	}
	require.EqualError(t, wrapped, direct.Error(), "validating the wrapped handler must see the same route table")
}

// TestFastPathDisabledByConfig checks the escape hatch returns the plain gorilla router.
func TestFastPathDisabledByConfig(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	cfg := config.TestConfig()
	cfg.Server.DisableFastRouter = true

	h := MakeHandler(svc, cfg, slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.IsType(t, &mux.Router{}, h)
}

// TestFastPathFallsBackWhenARouteIsAmbiguous checks the fail-safe: a conflicting route disables the fast path rather than
// panicking at startup or silently mis-routing.
func TestFastPathFallsBackWhenARouteIsAmbiguous(t *testing.T) {
	router := mux.NewRouter()
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	router.Handle("/api/v1/fleet/hosts/{id:[0-9]+}/software", noop).Methods("GET").Name("a")
	router.Handle("/api/v1/fleet/hosts/identifier/{identifier}", noop).Methods("GET").Name("b")

	h := newFastPathHandler(router, nil, config.TestConfig(), slog.New(slog.DiscardHandler))
	require.Same(t, router, h, "an ambiguous route should disable the fast path, not break the server")
}

func TestIsCanonicalPath(t *testing.T) {
	for _, c := range []struct {
		path string
		want bool
	}{
		{"/api/v1/fleet/config", true},
		{"/api/v1/fleet/results/", true},
		{"/", true},
		{"/api/v1//fleet/config", false},
		{"/api/v1/./fleet/config", false},
		{"/api/v1/fleet/../config", false},
		{"/api/v1/fleet/config//", false},
		{"", false},
		{"api/v1/fleet/config", false},
	} {
		require.Equalf(t, c.want, isCanonicalPath(c.path), "isCanonicalPath(%q)", c.path)
	}
}
