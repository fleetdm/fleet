package endpointer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	templateEcho := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tpl, _ := RouteTemplateFromContext(r.Context())
		_, _ = w.Write([]byte(tpl + " " + r.PathValue("id") + r.PathValue("name"))) //nolint:gosec // test echo, never rendered
	})
	r := NewRouter()
	for _, name := range []string{"outer", "inner"} {
		r.Use(func(_ Route, next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Add("X-Middleware", name)
				next.ServeHTTP(w, req)
			})
		})
	}
	r.Handle(Route{
		Name:     "get_host",
		Methods:  []string{http.MethodGet},
		Path:     "/api/_version_/fleet/hosts/{id:[0-9]+}",
		Versions: []string{"v1", "latest"},
	}, templateEcho)
	r.Handle(Route{Name: "delete_label", Methods: []string{http.MethodDelete}, Path: "/api/v1/fleet/hosts/{name}"}, templateEcho)
	r.Handle(Route{Name: "any_method", Path: "/api/any"}, templateEcho)
	r.Handle(Route{Name: "subtree", Methods: []string{http.MethodGet}, Path: "/api/tree/", Prefix: true}, templateEcho)
	r.Handle(Route{Name: "slash", Methods: []string{http.MethodGet}, Path: "/api/slash/"}, templateEcho)
	const hostTemplate = "/api/{fleetversion:(?:v1|latest)}/fleet/hosts/{id:[0-9]+}"
	assert.Panics(t, func() { r.Use(func(_ Route, next http.Handler) http.Handler { return next }) }, "Use after Handle")

	routes := r.Routes()
	require.Len(t, routes, 5)
	assert.Equal(t, "get_host", routes[0].Name)
	assert.Equal(t, hostTemplate, routes[0].Template())

	// Wrapping after registration must reach requests, inside the route middleware, with the template still in context.
	var wrapped []string
	r.WrapHandlers(func(rt Route, next http.Handler) http.Handler {
		wrapped = append(wrapped, rt.Name)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Route", rt.Name)
			next.ServeHTTP(w, req)
		})
	})
	assert.Equal(t, []string{"get_host", "delete_label", "any_method", "subtree", "slash"}, wrapped)

	for _, tc := range []struct {
		method, path string
		status       int
		route, body  string
		allow        string
	}{
		{http.MethodGet, "/api/latest/fleet/hosts/1", http.StatusOK, "get_host", hostTemplate + " 1", ""},
		{http.MethodHead, "/api/v1/fleet/hosts/1", http.StatusOK, "get_host", "", ""},
		{http.MethodGet, "/api/v9/fleet/hosts/1", http.StatusNotFound, "", "", ""},
		{http.MethodPost, "/api/v1/fleet/hosts/1", http.StatusMethodNotAllowed, "", "", "DELETE, GET, HEAD"},
		// A value that fails the constraint is a 404, or a 405 when another method's route takes it.
		{http.MethodGet, "/api/latest/fleet/hosts/abc", http.StatusNotFound, "", "", ""},
		{http.MethodGet, "/api/v1/fleet/hosts/abc", http.StatusMethodNotAllowed, "", "", "DELETE"},
		{http.MethodDelete, "/api/v1/fleet/hosts/a%2Fb", http.StatusOK, "delete_label", "/api/v1/fleet/hosts/{name} a/b", ""},
		// A route without methods serves every method.
		{http.MethodGet, "/api/any", http.StatusOK, "any_method", "/api/any ", ""},
		{http.MethodDelete, "/api/any", http.StatusOK, "any_method", "/api/any ", ""},
		{http.MethodGet, "/api/tree/a/b", http.StatusOK, "subtree", "/api/tree/ ", ""},
		{http.MethodGet, "/api/slash/", http.StatusOK, "slash", "/api/slash/ ", ""},
		{http.MethodGet, "/api/slash/a", http.StatusNotFound, "", "", ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			assert.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.route, rec.Header().Get("X-Route"))
			if tc.route != "" {
				assert.Equal(t, []string{"outer", "inner"}, rec.Header().Values("X-Middleware"))
			} else {
				assert.Empty(t, rec.Header().Values("X-Middleware"), "route middleware only runs for a matched route")
			}
			if tc.body != "" {
				assert.Equal(t, tc.body, rec.Body.String())
			}
			assert.Equal(t, tc.allow, rec.Header().Get("Allow"))
		})
	}
}

func TestRouterRejectsBadRoutes(t *testing.T) {
	t.Parallel()

	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, tc := range []struct {
		name      string
		routes    []Route
		wantPanic string
	}{
		{
			name: "unsupported constraint",
			routes: []Route{
				{Name: "bad", Methods: []string{http.MethodGet}, Path: "/api/{id:[0-9]{3}}"},
			},
			wantPanic: `route bad: unsupported constraint`,
		},
		{
			name: "duplicate route",
			routes: []Route{
				{Name: "first", Methods: []string{http.MethodGet}, Path: "/api/config"},
				{Name: "second", Methods: []string{http.MethodGet}, Path: "/api/config"},
			},
			wantPanic: `route second: "GET /api/config" is ambiguous`,
		},
		{
			name: "ambiguous route",
			routes: []Route{
				{Name: "host_software", Methods: []string{http.MethodGet}, Path: "/api/hosts/{id:[0-9]+}/software"},
				{Name: "host_by_identifier", Methods: []string{http.MethodGet}, Path: "/api/hosts/identifier/{identifier}"},
			},
			wantPanic: `route host_by_identifier: "GET /api/hosts/identifier/{identifier}" is ambiguous`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter()
			for _, rt := range tc.routes[:len(tc.routes)-1] {
				r.Handle(rt, noop)
			}
			defer func() {
				p := recover()
				require.NotNil(t, p)
				assert.Contains(t, p, tc.wantPanic)
			}()
			r.Handle(tc.routes[len(tc.routes)-1], noop)
		})
	}
}

// TestRouterDecodesPathValuesOnce runs StringFromRequest behind the router, the way request decoders read a name from the path.
func TestRouterDecodesPathValuesOnce(t *testing.T) {
	t.Parallel()

	r := NewRouter()
	r.Handle(Route{Name: "get_label", Methods: []string{http.MethodGet}, Path: "/api/labels/{name}"},
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			name, err := StringFromRequest(req, "name")
			assert.NoError(t, err)
			_, _ = w.Write([]byte(name)) //nolint:gosec // test echo, never rendered
		}))

	for _, tc := range []struct{ path, want string }{
		{"/api/labels/abc%2Fdef", "abc/def"},
		{"/api/labels/abc%252Fdef", "abc%2Fdef"},
		{"/api/labels/50%25off", "50%off"},
		{"/api/labels/a%20b", "a b"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.want, rec.Body.String())
		})
	}
}
