package endpointer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	templateEcho := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tpl, _ := RouteTemplateFromContext(r.Context())
		_, _ = w.Write([]byte(tpl))
	})
	newRouter := func() *Router {
		r := NewRouter()
		r.Handle(Route{
			Name:     "get_host",
			Methods:  []string{http.MethodGet},
			Path:     "/api/_version_/fleet/hosts/{id:[0-9]+}",
			Versions: []string{"v1", "latest"},
		}, templateEcho)
		r.Handle(Route{Name: "any_method", Path: "/api/any"}, templateEcho)
		return r
	}
	serve := func(r *Router, method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}
	const hostTemplate = "/api/{fleetversion:(?:v1|latest)}/fleet/hosts/{id:[0-9]+}"

	t.Run("puts the route template in the request context", func(t *testing.T) {
		rec := serve(newRouter(), http.MethodGet, "/api/v1/fleet/hosts/1")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, hostTemplate, rec.Body.String())
	})

	t.Run("records routes in registration order", func(t *testing.T) {
		routes := newRouter().Routes()
		require.Len(t, routes, 2)
		require.Equal(t, "get_host", routes[0].Name)
		require.Equal(t, hostTemplate, routes[0].Template())
		require.Equal(t, "any_method", routes[1].Name)
	})

	t.Run("a route without methods serves every method", func(t *testing.T) {
		r := newRouter()
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
			require.Equal(t, http.StatusOK, serve(r, method, "/api/any").Code, method)
		}
		require.Equal(t, http.StatusMethodNotAllowed, serve(r, http.MethodPost, "/api/v1/fleet/hosts/1").Code)
	})

	t.Run("WrapHandlers wraps every route and keeps the template", func(t *testing.T) {
		r := newRouter()
		var wrapped []string
		r.WrapHandlers(func(rt Route, next http.Handler) http.Handler {
			wrapped = append(wrapped, rt.Name)
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("X-Route", rt.Name)
				next.ServeHTTP(w, req)
			})
		})
		require.Equal(t, []string{"get_host", "any_method"}, wrapped)

		rec := serve(r, http.MethodGet, "/api/latest/fleet/hosts/1")
		require.Equal(t, "get_host", rec.Header().Get("X-Route"))
		require.Equal(t, hostTemplate, rec.Body.String())
	})

	t.Run("an invalid route panics at registration", func(t *testing.T) {
		require.Panics(t, func() {
			NewRouter().Handle(Route{Name: "bad", Methods: []string{http.MethodGet}, Path: "/api/{id:[0-9}"}, templateEcho)
		})
	})
}
