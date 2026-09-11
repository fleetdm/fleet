package service

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	activity_bootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	chart_bootstrap "github.com/fleetdm/fleet/v4/server/chart/bootstrap"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	acme_bootstrap "github.com/fleetdm/fleet/v4/server/mdm/acme/bootstrap"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
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
	h, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil,
		productionFeatureRoutes(t, svc))
	require.NoError(t, err)
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

// productionFeatureRoutes returns the same feature-route set cmd/fleet/serve.go passes to MakeHandler: the Android, activity,
// ACME, and chart bounded contexts. Their dependencies are zero valued because these tests only register routes and never
// serve them -- every handler is replaced before a request is made. Registering the real sets is what keeps the routing tests
// honest: with nil here they covered only the core table and missed the activity context's route template, which took the
// whole fast path down at startup.
func productionFeatureRoutes(t *testing.T, fleetSvc fleet.Service) []endpointer.HandlerRoutesFunc {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	conns := &platform_mysql.DBConnections{}
	// The endpointer panics on a nil auth middleware, and these endpoints are registered but never invoked.
	passthrough := func(next endpoint.Endpoint) endpoint.Endpoint { return next }

	_, activityRoutes := activity_bootstrap.New(conns, nil, nil, logger)
	_, acmeRoutes := acme_bootstrap.New(conns, nil, nil, logger)
	_, chartRoutes := chart_bootstrap.New(conns, nil, nil, logger)

	return []endpointer.HandlerRoutesFunc{
		android_service.GetRoutes(fleetSvc, nil),
		activityRoutes(passthrough),
		acmeRoutes(passthrough),
		chartRoutes(passthrough),
	}
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
			return nil // a route with no path is not reachable
		}
		methods, err := route.GetMethods()
		if err != nil || len(methods) == 0 {
			return nil
		}
		for _, method := range methods {
			samples = append(samples, sample{method: method, path: sampleRequestPath(tpl), route: route.GetName()})
		}
		return nil
	}))
	require.Greater(t, len(samples), 400, "expected the full route table to be walked")

	handler, err := newFastPathHandler(router, nil, config.TestConfig())
	require.NoError(t, err)
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
	handler, err := newFastPathHandler(router, nil, config.TestConfig())
	require.NoError(t, err)

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
	h, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.NoError(t, err)
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
	handler, err := newFastPathHandler(router, nil, config.TestConfig())
	require.NoError(t, err)

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

	handler, err := newFastPathHandler(router, middlewares, config.TestConfig())
	require.NoError(t, err)
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
			return nil
		}
		methods, err := route.GetMethods()
		if err != nil || len(methods) != 1 {
			return nil
		}
		if _, supported := supportedVarMatchers(tpl); !supported {
			return nil
		}
		key := unversionedKey(methods[0], tpl)
		registered[key] = struct{}{}
		for _, pattern := range expandToStdlibPatterns(tpl) {
			if conflictErr := tryRegister(fast, methods[0]+" "+pattern, noop); conflictErr != nil {
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
	h, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.NoError(t, err)
	router := h.(interface{ Router() *mux.Router }).Router()

	wrapped := apiendpoints.Validate(h)
	direct := apiendpoints.Validate(router)
	if direct == nil {
		require.NoError(t, wrapped)
		return
	}
	require.EqualError(t, wrapped, direct.Error(), "validating the wrapped handler must see the same route table")
}

// TestStdlibPatternsDedupesVersions covers the template a bounded context produces when it lists "latest" in its own version
// set: the endpointer appends "latest" a second time, and the repeated alternative must not become a repeated pattern.
func TestExpandToStdlibPatternsDedupesVersions(t *testing.T) {
	patterns := expandToStdlibPatterns("/api/{fleetversion:(?:v1|latest|latest)}/fleet/activities")
	require.Equal(t, []string{
		"/api/latest/fleet/activities",
		"/api/v1/fleet/activities",
	}, patterns)
}

// TestFastPathSurvivesFeatureRoutes is the regression test for the bounded contexts. MakeHandler is called with feature route
// functions the way cmd/fleet does, including one that repeats "latest" in its version set and one that re-registers a path the
// core handler already owns. Neither may knock the fast path out.
func TestFastPathSurvivesFeatureRoutes(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	noop := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })

	featureRoutes := []endpointer.HandlerRoutesFunc{
		// "latest" appears in the version list and is appended again by the endpointer, as the activity context does.
		func(r *mux.Router, _ []kithttp.ServerOption) {
			r.Handle("/api/{fleetversion:(?:v1|latest|latest)}/fleet/activities", noop).
				Methods("GET").Name("feature_activities")
		},
		// A second context claiming a path the core handler already registered.
		func(r *mux.Router, _ []kithttp.ServerOption) {
			r.Handle("/api/{fleetversion:(?:v1|2022-04|latest)}/fleet/config", noop).
				Methods("GET").Name("feature_duplicate_config")
		},
	}

	h, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, featureRoutes)
	require.NoError(t, err)
	require.IsType(t, &fastPathHandler{}, h, "feature routes must not disable the fast path")

	router := h.(*fastPathHandler).Router()
	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		name := route.GetName()
		route.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(name)) })
		return nil
	}))
	handler, err := newFastPathHandler(router, nil, config.TestConfig())
	require.NoError(t, err)
	require.IsType(t, &fastPathHandler{}, handler)

	// A duplicate registration resolves the way gorilla resolves it: first one registered wins.
	for _, path := range []string{"/api/latest/fleet/activities", "/api/v1/fleet/activities", "/api/latest/fleet/config"} {
		wantCode, wantBody, _ := responseFor(router, "GET", path)
		gotCode, gotBody, _ := responseFor(handler, "GET", path)
		require.Equalf(t, wantCode, gotCode, "status differs for GET %s", path)
		require.Equalf(t, wantBody, gotBody, "dispatch differs for GET %s", path)
	}
}

// TestFastPathErrorsWhenARouteIsAmbiguous pins the loud failure. A route the stdlib mux cannot disambiguate is a programming
// error in a compile-time-fixed route table, so it has to stop startup rather than silently switch the fast path off, and the
// message has to tell the developer which route broke it and what to edit.
func TestFastPathErrorsWhenARouteIsAmbiguous(t *testing.T) {
	router := mux.NewRouter()
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	router.Handle("/api/v1/fleet/hosts/{id:[0-9]+}/software", noop).Methods("GET").Name("get_host_software")
	router.Handle("/api/v1/fleet/hosts/identifier/{identifier}", noop).Methods("GET").Name("get_host_by_identifier")

	h, err := newFastPathHandler(router, nil, config.TestConfig())
	require.Nil(t, h)
	require.Error(t, err, "an ambiguous route must fail startup, not degrade quietly")
	require.Contains(t, err.Error(), "get_host_by_identifier", "the error must name the offending route")
	require.Contains(t, err.Error(), "fastPathExcluded", "the error must name the fix")
}

// TestTryRegisterRepanicsOnNonConflict checks the narrowing: only a pattern-conflict panic becomes an error. A malformed
// pattern is a different bug, and reporting it as an ambiguity would send the reader to edit fastPathExcluded for no reason.
func TestTryRegisterRepanicsOnNonConflict(t *testing.T) {
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	t.Run("conflict becomes an error", func(t *testing.T) {
		m := http.NewServeMux()
		require.NoError(t, tryRegister(m, "GET /a/identifier/{identifier}", noop))
		err := tryRegister(m, "GET /a/{id}/software", noop)
		require.ErrorContains(t, err, "fastPathExcluded")
	})

	t.Run("malformed pattern still panics", func(t *testing.T) {
		m := http.NewServeMux()
		defer func() {
			r := recover()
			require.NotNil(t, r, "a malformed pattern must not be swallowed as an ambiguity")
			panicErr, ok := r.(error)
			require.True(t, ok)
			require.Contains(t, panicErr.Error(), "parsing", "the stdlib diagnosis has to survive")
			require.NotContains(t, panicErr.Error(), "fastPathExcluded", "and must not be relabelled as a route conflict")
		}()
		_ = tryRegister(m, "GET /a/{bad", noop)
	})
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
