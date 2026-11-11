package endpointer

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteTemplate(t *testing.T) {
	cases := []struct {
		name     string
		route    Route
		expected string
	}{
		{
			name:     "versioned",
			route:    Route{Path: "/api/_version_/fleet/hosts/{id:[0-9]+}", Versions: []string{"v1", "2022-04", "latest"}},
			expected: "/api/{fleetversion:(?:v1|2022-04|latest)}/fleet/hosts/{id:[0-9]+}",
		},
		{
			name:     "unversioned",
			route:    Route{Path: "/api/fleet/orbit/ping", Versions: []string{"v1", "latest"}},
			expected: "/api/fleet/orbit/ping",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, c.route.Template())
		})
	}
}

// TestRouteTableRecordsEveryRegistration checks that each registration function records the routes it puts on gorilla, so code
// that reads the table sees the same route list as code that walks the router.
func TestRouteTableRecordsEveryRegistration(t *testing.T) {
	r := mux.NewRouter()
	versions := []string{"v1", "2022-04"}
	registry := NewHandlerRegistry()
	ep := &CommonEndpointer[any]{Router: r, Versions: versions, HandlerRegistry: registry}

	ep.HandleHTTPHandler("/api/_version_/fleet/hosts/{id:[0-9]+}", http.NotFoundHandler(), http.MethodGet)
	ep.EndingAtVersion("v1").WithAltPaths("/api/_version_/fleet/computers").HandleHTTPHandler("/api/_version_/fleet/devices",
		http.NotFoundHandler(), http.MethodPost)
	ep.UsePathPrefix().HandleHTTPHandler("/api/_version_/fleet/files/", http.NotFoundHandler(), http.MethodGet)
	RegisterDeprecatedPathAliases(r, versions, registry, []DeprecatedPathAlias{{
		Method:          http.MethodGet,
		PrimaryPath:     "/api/_version_/fleet/hosts/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/machines/{id:[0-9]+}"},
	}})
	RegisterRemovedPaths(r, versions, slog.New(slog.DiscardHandler), []RemovedPath{{
		Path:    "/api/_version_/fleet/packs/{id}/scheduled",
		Methods: []string{http.MethodGet, http.MethodPost},
		Overlap: RemovedPathOverlap{Method: http.MethodGet, RouteTemplate: "/api/_version_/fleet/packs/{id}/{segment}"},
		Message: func(string) string { return "gone" },
	}})

	type registered struct {
		name, template string
		methods        []string
	}
	var walked []registered
	require.NoError(t, r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := route.GetPathTemplate()
		require.NoError(t, err)
		methods, err := route.GetMethods()
		require.NoError(t, err)
		walked = append(walked, registered{route.GetName(), tpl, methods})
		return nil
	}))

	var recorded []registered
	for _, rt := range RoutesOf(r).Routes() {
		recorded = append(recorded, registered{rt.Name, rt.Template(), rt.Methods})
	}
	require.Len(t, recorded, 7)
	assert.Equal(t, walked, recorded)

	assert.Empty(t, RoutesOf(mux.NewRouter()).Routes(), "each router has its own table")
}
