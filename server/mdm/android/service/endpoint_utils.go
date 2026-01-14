package service

import (
	"context"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return eu.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response interface{}) error {
			return json.MarshalWrite(w, response, jsontext.WithIndent("  "))
		},
		nil, // no domain-specific error encoder
	)
}

func makeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.UnmarshalRead(body, req)
	}, nil, nil, nil, nil)
}

// handlerFunc is the handler function type for Android service endpoints.
type handlerFunc func(ctx context.Context, request any, svc android.Service) fleet.Errorer

// Compile-time check to ensure that androidEndpointer implements Endpointer.
var _ eu.Endpointer[handlerFunc] = &androidEndpointer{}

type androidEndpointer struct {
	svc android.Service
}

func (e *androidEndpointer) CallHandlerFunc(f handlerFunc, ctx context.Context, request any,
	svc any) (platform_http.Errorer, error) {
	return f(ctx, request, svc.(android.Service)), nil
}

func (e *androidEndpointer) Service() any {
	return e.svc
}

func newUserAuthenticatedEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &androidEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return auth.AuthenticatedUser(fleetSvc, next)
		},
		Router:   r,
		Versions: versions,
	}
}

func newNoAuthEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &androidEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return auth.UnauthenticatedRequest(fleetSvc, next)
		},
		Router:   r,
		Versions: versions,
	}
}
