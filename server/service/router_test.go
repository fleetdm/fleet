package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/acl/chartacl"
	activity_bootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	chart_bootstrap "github.com/fleetdm/fleet/v4/server/chart/bootstrap"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	acme_bootstrap "github.com/fleetdm/fleet/v4/server/mdm/acme/bootstrap"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	notifications_bootstrap "github.com/fleetdm/fleet/v4/server/notifications/bootstrap"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newAPIHandler builds the real API router the way cmd/fleet does.
func newAPIHandler(t *testing.T, cfg config.FleetConfig, featureRoutes []endpointer.HandlerRoutesFunc) *endpointer.Router {
	t.Helper()
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)

	h, err := MakeHandler(svc, cfg, slog.New(slog.DiscardHandler), limitStore, nil, nil, featureRoutes)
	require.NoError(t, err)
	router, ok := h.(*endpointer.Router)
	require.True(t, ok, "MakeHandler should return the router")
	return router
}

// productionFeatureRoutes returns the same feature-route set cmd/fleet/serve.go passes to MakeHandler.
func productionFeatureRoutes(t *testing.T, fleetSvc fleet.Service) []endpointer.HandlerRoutesFunc {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	conns := &platform_mysql.DBConnections{}
	// The endpointer panics on a nil auth middleware, and these endpoints are registered but never invoked.
	passthrough := func(next endpoint.Endpoint) endpoint.Endpoint { return next }

	_, activityRoutes := activity_bootstrap.New(conns, nil, nil, logger)
	_, acmeRoutes := acme_bootstrap.New(conns, nil, nil, logger)
	_, chartRoutes := chart_bootstrap.New(conns, nil, nil, chartacl.ExpandPlatform, logger)
	_, notificationsRoutes := notifications_bootstrap.New(conns, nil, logger)

	return FeatureRoutes(
		android_service.GetRoutes(fleetSvc, nil),
		activityRoutes(passthrough),
		notificationsRoutes(passthrough),
		acmeRoutes(passthrough),
		chartRoutes(passthrough),
	)
}

// newRouterForTest builds the production router and replaces every handler with one that writes the route it belongs to and
// the path variables it resolved. It fails the test when the route template in context is not the route's own, which is what
// API-only authorization matches on.
func newRouterForTest(t *testing.T, cfg config.FleetConfig) *endpointer.Router {
	t.Helper()
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	router := newAPIHandler(t, cfg, productionFeatureRoutes(t, svc))

	router.WrapHandlers(func(rt endpointer.Route, _ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tpl, _ := endpointer.RouteTemplateFromContext(r.Context())
			assert.Equalf(t, rt.Template(), tpl, "route template in context for %s %s", r.Method, r.URL.Path)
			var vars []string
			for _, m := range reSimpleVar.FindAllStringSubmatch(reNumVar.ReplaceAllStringFunc(rt.Path, stripConstraint), -1) {
				v, _ := platform_http.PathVar(r, m[1])
				vars = append(vars, m[1]+"="+v)
			}
			_, _ = w.Write([]byte(rt.Name + " " + strings.Join(vars, ","))) //nolint:gosec // test echo, never rendered
		})
	})
	return router
}

// stripConstraint turns "{id:[0-9]+}" into "{id}".
func stripConstraint(v string) string {
	name, _, _ := strings.Cut(v, ":")
	return name + "}"
}

func serve(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(method, path, nil))
	return rr
}

// TestRouterServesEveryRoute sends one request per registered route and method, and requires the route itself to answer with
// every path variable resolved. Anything else means another route shadows it or a variable got lost.
func TestRouterServesEveryRoute(t *testing.T) {
	router := newRouterForTest(t, config.TestConfig())

	var compared int
	for _, rt := range router.Routes() {
		_, path, err := routeTestRequest(rt)
		require.NoError(t, err)
		if path == "" {
			continue
		}
		for _, method := range rt.Methods {
			got := serve(router, method, path)
			// assert rather than require: a systemic break reports every affected route instead of stopping at the first.
			if !assert.Equalf(t, http.StatusOK, got.Code, "%s %s (route %s)", method, path, rt.Name) {
				continue
			}
			name, vars, _ := strings.Cut(got.Body.String(), " ")
			assert.Equalf(t, rt.Name, name, "%s %s was served by another route", method, path)
			for v := range strings.SplitSeq(vars, ",") {
				assert.Falsef(t, strings.HasSuffix(v, "="), "%s %s (route %s) lost path variable %s", method, path, rt.Name, v)
			}
			compared++
		}
	}
	require.Greater(t, compared, 400, "expected the full route table")
}

// TestRouterEdgeCases covers the production routes where matching is not a plain lookup: the removed-path wildcards next to
// live routes, and the deliberate 405 for a wrong method with an invalid id. TestRouter in the endpointer covers the rest.
func TestRouterEdgeCases(t *testing.T) {
	router := newRouterForTest(t, config.TestConfig())

	cases := []struct {
		name         string
		method, path string
		wantStatus   int
		// wantBody is the route name and resolved variables when a route answers.
		wantBody  string
		wantAllow string
	}{
		{
			name: "live resend wins over the removed spelling", method: "POST",
			path:       "/api/latest/fleet/hosts/1/configuration_profiles/resend/resend",
			wantStatus: 200, wantBody: "post_hosts__host_id:[0-9]+__configuration_profiles__profile_uuid__resend host_id=1,profile_uuid=resend",
		},
		{
			name: "removed route behind a live route whose constraint fails", method: "GET",
			path:       "/api/latest/fleet/hosts/identifier/device_mapping",
			wantStatus: 200, wantBody: "get_hosts_identifier__identifier_ identifier_segment=identifier,identifier=device_mapping",
		},
		{
			// Deliberate: the ServeMux answers a wrong method before any route constraint runs, so the invalid id stays a 405.
			name: "invalid id on a path only other methods serve", method: "POST", path: "/api/latest/fleet/certificates/abc",
			wantStatus: 405, wantAllow: "DELETE, GET, HEAD",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := serve(router, c.method, c.path)
			require.Equal(t, c.wantStatus, got.Code)
			if c.wantBody != "" {
				require.Equal(t, c.wantBody, got.Body.String())
			}
			if c.wantAllow != "" {
				require.Equal(t, c.wantAllow, got.Header().Get("Allow"))
			}
		})
	}
}

// TestRouterSpanNamesFeedTheTracingTierRegistry pins the span-naming contract end to end.
func TestRouterSpanNamesFeedTheTracingTierRegistry(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	cfg := config.TestConfig()
	cfg.Logging.TracingEnabled = true
	cfg.Logging.TracingType = "opentelemetry"
	require.True(t, cfg.OTELEnabled())
	router := newRouterForTest(t, cfg)

	registry := tracing.NewRegistry()
	RegisterTracingTiers(registry)

	cases := []struct {
		name         string
		method, path string
		wantSpan     string
		wantTier     tracing.Tier
	}{
		{
			name: "literal agent route", method: "POST", path: "/api/osquery/distributed/write",
			wantSpan: "POST /api/osquery/distributed/write", wantTier: tracing.TierHighVolume,
		},
		{
			name: "version and path variable are named by their placeholders", method: "GET", path: "/api/latest/fleet/hosts/1",
			wantSpan: "GET /api/_version_/fleet/hosts/{id}", wantTier: tracing.TierStandard,
		},
		{
			name: "HEAD on a GET route takes the GET tier", method: "HEAD", path: "/api/latest/fleet/config",
			wantSpan: "HEAD /api/_version_/fleet/config", wantTier: tracing.TierStandard,
		},
		{
			name: "constrained variable drops its regex", method: "PATCH", path: "/api/latest/fleet/fleets/1/secrets",
			wantSpan: "PATCH /api/_version_/fleet/fleets/{fleet_id}/secrets", wantTier: tracing.TierAlways,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recorder.Reset()
			require.Equal(t, http.StatusOK, serve(router, c.method, c.path).Code)

			ended := recorder.Ended()
			require.Len(t, ended, 1, "one span per request")
			require.Equal(t, c.wantSpan, ended[0].Name())
			// Both spellings of the route have to agree, or a reader correlating a trace to the route keyed server metrics
			// has two routes for one request.
			require.Equal(t, []string{strings.TrimPrefix(c.wantSpan, c.method+" ")}, spanRoutes(ended[0]))

			tier, found := registry.Lookup(c.wantSpan)
			if c.wantTier != tracing.TierAlways {
				require.Truef(t, found, "%q must resolve in the tier registry, otherwise it samples at 100%%", c.wantSpan)
			}
			require.Equal(t, c.wantTier, tier)
		})
	}

	t.Run("a request no route matches has no span", func(t *testing.T) {
		recorder.Reset()
		require.Equal(t, http.StatusNotFound, serve(router, "GET", "/api/latest/fleet/nope").Code)
		require.Equal(t, http.StatusNotFound, serve(router, "GET", "/api/latest/fleet/certificates/abc").Code)
		require.Empty(t, recorder.Ended())
	})
}

// spanRoutes returns every http.route attribute on the span. More than one means the instrumentation disagrees with itself.
func spanRoutes(span sdktrace.ReadOnlySpan) []string {
	var routes []string
	for _, attr := range span.Attributes() {
		if string(attr.Key) == "http.route" {
			routes = append(routes, attr.Value.AsString())
		}
	}
	return routes
}

// TestAPIOnlyCatalogThroughRouter routes a request to every catalog entry and runs the API-only check behind the router. A user
// granted that entry must pass and a user granted another one must not, which fails if the router hands authorization a route
// template that maps to the wrong catalog entry, or none.
func TestAPIOnlyCatalogThroughRouter(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	router := newAPIHandler(t, config.TestConfig(), productionFeatureRoutes(t, svc))

	var grant fleet.APIEndpointRef
	allowed := func(context.Context, any) (any, error) { return nil, nil }
	router.WrapHandlers(func(_ endpointer.Route, _ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := &fleet.User{APIOnly: true, APIEndpoints: []fleet.APIEndpointRef{grant}}
			ctx := viewer.NewContext(kithttp.PopulateRequestContext(r.Context(), r), viewer.Viewer{User: user})
			if _, err := auth.APIOnlyEndpointCheck(allowed)(ctx, nil); err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
	})

	unrelated := fleet.APIEndpointRef{Method: http.MethodGet, Path: "/api/v1/fleet/version"}
	placeholder := regexp.MustCompile(`:[a-zA-Z_]+`)
	var checked int
	for _, entry := range apiendpoints.GetAPIEndpoints() {
		if strings.Contains(entry.Path, "/scim/") {
			continue // served by the SCIM handler, not this router
		}
		path := placeholder.ReplaceAllString(entry.Path, "1")
		if serve(router, entry.Method, path).Code == http.StatusNotFound {
			path = strings.Replace(path, "/api/v1/", "/api/latest/", 1) // a route that starts after v1
		}
		grant = fleet.APIEndpointRef{Method: entry.Method, Path: entry.Path}
		assert.Equalf(t, http.StatusOK, serve(router, entry.Method, path).Code, "%s %s granted itself", entry.Method, path)
		if grant != unrelated {
			grant = unrelated
			assert.Equalf(t, http.StatusForbidden, serve(router, entry.Method, path).Code, "%s %s granted another endpoint",
				entry.Method, path)
		}
		checked++
	}
	require.Greater(t, checked, 200, "expected the whole catalog")
}
