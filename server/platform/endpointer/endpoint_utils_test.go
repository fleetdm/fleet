package endpointer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// testHandlerFunc is a handler function type used for testing.
type testHandlerFunc func(ctx context.Context, request any) (platform_http.Errorer, error)

func TestCustomMiddlewareAfterAuth(t *testing.T) {
	var (
		i                = 0
		beforeIndex      = 0
		authIndex        = 0
		afterFirstIndex  = 0
		afterSecondIndex = 0
	)
	beforeAuthMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			i++
			beforeIndex = i
			return next(ctx, req)
		}
	}

	authMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			i++
			authIndex = i
			if authctx, ok := authz_ctx.FromContext(ctx); ok {
				authctx.SetChecked()
			}
			return next(ctx, req)
		}
	}

	afterAuthMiddlewareFirst := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			i++
			afterFirstIndex = i
			return next(ctx, req)
		}
	}
	afterAuthMiddlewareSecond := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			i++
			afterSecondIndex = i
			return next(ctx, req)
		}
	}

	r := mux.NewRouter()
	ce := &CommonEndpointer[testHandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (request any, err error) {
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i any) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthMiddleware: authMiddleware,
		CustomMiddleware: []endpoint.Middleware{
			beforeAuthMiddleware,
		},
		CustomMiddlewareAfterAuth: []endpoint.Middleware{
			afterAuthMiddlewareFirst,
			afterAuthMiddlewareSecond,
		},
		Router: r,
	}
	ce.handleEndpoint("/", func(ctx context.Context, request any) (platform_http.Errorer, error) {
		fmt.Printf("handler\n")
		return nopResponse{}, nil
	}, nil, "GET")

	s := httptest.NewServer(r)
	t.Cleanup(func() {
		s.Close()
	})

	req, err := http.NewRequest("GET", s.URL+"/", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		resp.Body.Close()
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, beforeIndex)
	require.Equal(t, 2, authIndex)
	require.Equal(t, 3, afterFirstIndex)
	require.Equal(t, 4, afterSecondIndex)
}

type nopRequest struct{}

type nopResponse struct{}

func (n nopResponse) Error() error {
	return nil
}

type nopEP struct{}

func (n nopEP) CallHandlerFunc(f testHandlerFunc, ctx context.Context, request any, svc any) (platform_http.Errorer, error) {
	return f(ctx, request)
}

func (n nopEP) Service() any {
	return nil
}

func TestRegisterDeprecatedPathAliases(t *testing.T) {
	// Set up a router and register a primary endpoint via CommonEndpointer.
	r := mux.NewRouter()
	registry := NewHandlerRegistry()
	versions := []string{"v1", "2022-04"}

	authMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if authctx, ok := authz_ctx.FromContext(ctx); ok {
				authctx.SetChecked()
			}
			return next(ctx, req)
		}
	}

	ce := &CommonEndpointer[testHandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (request any, err error) {
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i any) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthMiddleware:  authMiddleware,
		Router:          r,
		Versions:        versions,
		HandlerRegistry: registry,
	}

	// Register the primary endpoint.
	ce.GET("/api/_version_/fleet/fleets", func(ctx context.Context, request any) (platform_http.Errorer, error) {
		return nopResponse{}, nil
	}, nil)

	// Register a deprecated alias for it.
	RegisterDeprecatedPathAliases(r, versions, registry, []DeprecatedPathAlias{
		{
			Method:          "GET",
			PrimaryPath:     "/api/_version_/fleet/fleets",
			DeprecatedPaths: []string{"/api/_version_/fleet/teams"},
		},
	})

	s := httptest.NewServer(r)
	t.Cleanup(s.Close)

	// Both the primary and deprecated paths should return 200.
	for _, path := range []string{"/api/v1/fleet/fleets", "/api/v1/fleet/teams", "/api/latest/fleet/teams"} {
		resp, err := http.Get(s.URL + path)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "path %s should return 200", path)
	}
}

func TestLogDeprecatedPathAlias(t *testing.T) {
	// Without deprecated path info in context, LogDeprecatedPathAlias is a no-op.
	lc := &logging.LoggingContext{}
	ctx := logging.NewContext(context.Background(), lc)
	ctx2 := LogDeprecatedPathAlias(ctx, nil)
	require.Equal(t, ctx, ctx2, "should return same context when no deprecated path info")
	require.Empty(t, lc.Extras)

	// With deprecated path info, it should set warn level and extras.
	ctx = context.WithValue(ctx, deprecatedPathInfoKey{}, deprecatedPathInfo{
		deprecatedPath: "/api/_version_/fleet/teams",
		primaryPath:    "/api/_version_/fleet/fleets",
	})
	LogDeprecatedPathAlias(ctx, nil)

	// Extras is a flat []interface{} of key-value pairs.
	require.Len(t, lc.Extras, 4) // "deprecated_path", value, "deprecation_warning", value
	require.Equal(t, "deprecated_path", lc.Extras[0])
	require.Equal(t, "/api/_version_/fleet/teams", lc.Extras[1])
	require.Equal(t, "deprecation_warning", lc.Extras[2])
	require.Contains(t, lc.Extras[3], "deprecated")

	// ForceLevel should be set to Warn.
	require.NotNil(t, lc.ForceLevel)
	require.Equal(t, slog.LevelWarn, *lc.ForceLevel)
}

func TestRegisterDeprecatedPathAliasesPanicsOnMissing(t *testing.T) {
	r := mux.NewRouter()
	registry := NewHandlerRegistry()
	versions := []string{"v1"}

	require.Panics(t, func() {
		RegisterDeprecatedPathAliases(r, versions, registry, []DeprecatedPathAlias{
			{
				Method:          "GET",
				PrimaryPath:     "/api/_version_/fleet/nonexistent",
				DeprecatedPaths: []string{"/api/_version_/fleet/old"},
			},
		})
	})
}
