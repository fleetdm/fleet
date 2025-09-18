package endpoint_utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

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

	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
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
	ce := &CommonEndpointer[HandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface interface{}) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (request interface{}, err error) {
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i interface{}) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthFunc: authFunc,
		CustomMiddleware: []endpoint.Middleware{
			beforeAuthMiddleware,
		},
		CustomMiddlewareAfterAuth: []endpoint.Middleware{
			afterAuthMiddlewareFirst,
			afterAuthMiddlewareSecond,
		},
		Router: r,
	}
	ce.handleEndpoint("/", func(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

func (n nopEP) CallHandlerFunc(f HandlerFunc, ctx context.Context, request interface{}, svc interface{}) (fleet.Errorer, error) {
	return nopResponse{}, nil
}

func (n nopEP) Service() interface{} {
	return nil
}
