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
		_, _ = w.Write([]byte(tpl))
	})
	r := NewRouter()
	r.Handle(Route{
		Name:     "get_host",
		Methods:  []string{http.MethodGet},
		Path:     "/api/_version_/fleet/hosts/{id:[0-9]+}",
		Versions: []string{"v1", "latest"},
	}, templateEcho)
	r.Handle(Route{Name: "any_method", Path: "/api/any"}, templateEcho)
	const hostTemplate = "/api/{fleetversion:(?:v1|latest)}/fleet/hosts/{id:[0-9]+}"

	routes := r.Routes()
	require.Len(t, routes, 2)
	assert.Equal(t, "get_host", routes[0].Name)
	assert.Equal(t, hostTemplate, routes[0].Template())
	assert.Equal(t, "any_method", routes[1].Name)

	// Wrapping after registration must reach requests, and the template must stay in context.
	var wrapped []string
	r.WrapHandlers(func(rt Route, next http.Handler) http.Handler {
		wrapped = append(wrapped, rt.Name)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Route", rt.Name)
			next.ServeHTTP(w, req)
		})
	})
	assert.Equal(t, []string{"get_host", "any_method"}, wrapped)

	for _, tc := range []struct {
		method, path string
		status       int
		route, body  string
	}{
		{http.MethodGet, "/api/latest/fleet/hosts/1", http.StatusOK, "get_host", hostTemplate},
		{http.MethodPost, "/api/v1/fleet/hosts/1", http.StatusMethodNotAllowed, "", ""},
		// A route without methods serves every method.
		{http.MethodGet, "/api/any", http.StatusOK, "any_method", "/api/any"},
		{http.MethodDelete, "/api/any", http.StatusOK, "any_method", "/api/any"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			assert.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.route, rec.Header().Get("X-Route"))
			if tc.body != "" {
				assert.Equal(t, tc.body, rec.Body.String())
			}
		})
	}

	// gorilla keeps a route it rejected but never matches it, so registration must fail loudly instead.
	assert.Panics(t, func() {
		NewRouter().Handle(Route{Name: "bad", Methods: []string{http.MethodGet}, Path: "/api/{id:[0-9}"}, templateEcho)
	})
}
