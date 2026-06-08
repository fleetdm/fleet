package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authzctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// muxVersionSegment is the gorilla/mux route template version segment that
// RouteTemplateRequestFunc would extract from a real mux router.
const muxVersionSegment = "/api/{fleetversion:(?:v1|2022-04|latest)}/"

// testCatalogEndpoints is the minimal set of endpoints used across tests.
var testCatalogEndpoints = []fleet.APIEndpoint{
	fleet.NewAPIEndpointFromTpl("GET", "/api/v1/fleet/hosts"),
	fleet.NewAPIEndpointFromTpl("GET", "/api/v1/fleet/hosts/:id"),
	fleet.NewAPIEndpointFromTpl("POST", "/api/v1/fleet/scripts/run"),
}

// testIsInCatalog builds a fingerprint set from testCatalogEndpoints and
// returns an isInCatalog func suitable for injection into apiOnlyEndpointCheck.
func testIsInCatalog() func(string) bool {
	set := make(map[string]struct{}, len(testCatalogEndpoints))
	for _, ep := range testCatalogEndpoints {
		set[ep.Fingerprint()] = struct{}{}
	}
	return func(fp string) bool {
		_, ok := set[fp]
		return ok
	}
}

// muxTemplate returns a gorilla/mux route template for the given path suffix, simulating
// what RouteTemplateRequestFunc would extract from mux.CurrentRoute(r).GetPathTemplate().
func muxTemplate(pathSuffix string) string {
	return muxVersionSegment + pathSuffix
}

func TestAPIOnlyEndpointCheck(t *testing.T) {
	newNext := func() (func(context.Context, any) (any, error), *bool) {
		called := false
		fn := func(ctx context.Context, request any) (any, error) {
			called = true
			return nil, nil
		}
		return fn, &called
	}

	newEndpoint := func(next func(context.Context, any) (any, error)) func(context.Context, any) (any, error) {
		return apiOnlyEndpointCheck(testIsInCatalog(), next)
	}

	ctxWithMethod := func(method, tpl string) context.Context {
		ctx := context.Background()
		ctx = context.WithValue(ctx, kithttp.ContextKeyRequestMethod, method)
		ctx = eu.WithRouteTemplate(ctx, tpl)
		return ctx
	}

	t.Run("non-api-only user always passes through", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{APIOnly: false}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("no viewer in context passes through", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts"))
		// no viewer set

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user, endpoint in catalog, no restrictions", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: nil,
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user, empty APIEndpoints slice treated same as nil (no restrictions)", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: []fleet.APIEndpointRef{}, // empty, not nil
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user, endpoint with placeholder in catalog, no restrictions", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts/{id:[0-9]+}"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: nil,
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user with restrictions, endpoint not in catalog", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/secret_admin_endpoint"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			// Non-empty restrictions force the middleware to run; the route
			// is not in the catalog, so the catalog check rejects it.
			APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)
		var permErr *fleet.PermissionError
		require.ErrorAs(t, err, &permErr)
	})

	t.Run("api-only user with restrictions, missing route template in context is rejected", func(t *testing.T) {
		next, called := newNext()
		// routeTemplateKey deliberately not set (simulates RouteTemplateRequestFunc failure).
		ctx := context.Background()
		ctx = context.WithValue(ctx, kithttp.ContextKeyRequestMethod, "GET")
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)
		var permErr *fleet.PermissionError
		require.ErrorAs(t, err, &permErr)
	})

	t.Run("api-only user with restrictions, missing method and template are both rejected", func(t *testing.T) {
		next, called := newNext()
		// Neither method nor template set — empty fingerprint never matches catalog.
		ctx := context.Background()
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)
		var permErr *fleet.PermissionError
		require.ErrorAs(t, err, &permErr)
	})

	t.Run("api-only user, method normalization is case-insensitive", func(t *testing.T) {
		// Lower-case method must normalize to the same fingerprint as upper-case
		// when matching against the catalog and the user's allow-list.
		next, called := newNext()
		ctx := ctxWithMethod("get", muxTemplate("fleet/hosts")) // lower-case
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user with restrictions, rejection marks authz context as checked", func(t *testing.T) {
		// Ensures authzcheck middleware does not emit a spurious "Missing
		// authorization check" log when we deny an api_only user.
		next, called := newNext()
		ac := &authzctx.AuthorizationContext{}
		ctx := authzctx.NewContext(context.Background(), ac)
		ctx = context.WithValue(ctx, kithttp.ContextKeyRequestMethod, "GET")
		ctx = eu.WithRouteTemplate(ctx, muxTemplate("fleet/secret_admin_endpoint"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly:      true,
			APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)
		require.True(t, ac.Checked(), "authz context must be marked checked on denial")
	})

	t.Run("api-only user with restrictions, accessing allowed endpoint", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			APIEndpoints: []fleet.APIEndpointRef{
				{Method: "GET", Path: "/api/v1/fleet/hosts"},
			},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user with restrictions, accessing allowed placeholder endpoint", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/hosts/{id:[0-9]+}"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			// Stored path uses colon-prefix style as in the YAML catalog.
			APIEndpoints: []fleet.APIEndpointRef{
				{Method: "GET", Path: "/api/v1/fleet/hosts/:id"},
			},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})

	t.Run("api-only user with restrictions, accessing disallowed endpoint", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("POST", muxTemplate("fleet/scripts/run"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			APIEndpoints: []fleet.APIEndpointRef{
				{Method: "GET", Path: "/api/v1/fleet/hosts"},
			},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)

		var permErr *fleet.PermissionError
		require.ErrorAs(t, err, &permErr)
	})

	t.Run("api-only user, allow-list entry for non-catalog endpoint is still denied", func(t *testing.T) {
		// The catalog check runs before the allow-list check; an explicit allow entry
		// must not grant access to an endpoint that is not in the catalog.
		next, called := newNext()
		ctx := ctxWithMethod("GET", muxTemplate("fleet/secret_admin_endpoint"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			APIEndpoints: []fleet.APIEndpointRef{
				{Method: "GET", Path: "/api/v1/fleet/secret_admin_endpoint"},
			},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)
		var permErr *fleet.PermissionError
		require.ErrorAs(t, err, &permErr)
	})

	t.Run("api-only user, wrong method for catalog endpoint is rejected at catalog step", func(t *testing.T) {
		// POST /fleet/hosts is not in the catalog (only GET is), so the catalog check rejects it.
		next, called := newNext()
		ctx := ctxWithMethod("POST", muxTemplate("fleet/hosts"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			APIEndpoints: []fleet.APIEndpointRef{
				{Method: "GET", Path: "/api/v1/fleet/hosts"},
			},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.Error(t, err)
		require.False(t, *called)
		var permErr *fleet.PermissionError
		require.ErrorAs(t, err, &permErr)
	})

	t.Run("api-only user with multiple allowed endpoints, accessing one of them", func(t *testing.T) {
		next, called := newNext()
		ctx := ctxWithMethod("POST", muxTemplate("fleet/scripts/run"))
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			APIOnly: true,
			APIEndpoints: []fleet.APIEndpointRef{
				{Method: "GET", Path: "/api/v1/fleet/hosts"},
				{Method: "POST", Path: "/api/v1/fleet/scripts/run"},
			},
		}})

		_, err := newEndpoint(next)(ctx, nil)
		require.NoError(t, err)
		require.True(t, *called)
	})
}

func TestRouteTemplateRequestFunc(t *testing.T) {
	// Register a route and route the request through mux so mux.CurrentRoute
	// returns a non-nil value, mirroring what happens in production.
	newServedRequest := func(t *testing.T, routeTpl, reqPath string) (context.Context, bool) {
		t.Helper()
		var (
			got      context.Context
			wasMatch bool
		)
		r := mux.NewRouter()
		r.HandleFunc(routeTpl, func(_ http.ResponseWriter, req *http.Request) {
			wasMatch = true
			got = RouteTemplateRequestFunc(req.Context(), req)
		}).Methods("GET")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", reqPath, nil)
		r.ServeHTTP(rec, req)
		return got, wasMatch
	}

	t.Run("stores the matched route template", func(t *testing.T) {
		ctx, matched := newServedRequest(t, "/api/v1/fleet/hosts/{id:[0-9]+}", "/api/v1/fleet/hosts/42")
		require.True(t, matched, "expected route to be matched")
		tpl, ok := eu.RouteTemplateFromContext(ctx)
		require.True(t, ok, "route template must be stored in context")
		require.Equal(t, "/api/v1/fleet/hosts/{id:[0-9]+}", tpl)
	})

	t.Run("no matched route leaves context unchanged", func(t *testing.T) {
		// Call RouteTemplateRequestFunc directly with a request that never went
		// through a mux router, so mux.CurrentRoute returns nil.
		req := httptest.NewRequest("GET", "/whatever", nil)
		ctx := context.Background()
		got := RouteTemplateRequestFunc(ctx, req)
		_, ok := eu.RouteTemplateFromContext(got)
		require.False(t, ok, "no route template should be stored when no route is matched")
	})
}
