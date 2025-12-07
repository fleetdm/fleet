package service

import (
	"context"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	eu "github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// ActivityFunc is the handler function signature for activity service endpoints.
type ActivityFunc func(ctx context.Context, request any, svc activity.Service) platform_http.Errorer

func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	return eu.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response any) error {
			return json.MarshalWrite(w, response, jsontext.WithIndent("  "))
		},
	)
}

func makeDecoder(iface any) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.UnmarshalRead(body, req)
	}, nil, nil, nil)
}

// Compile-time check to ensure that endpointer implements Endpointer.
var _ eu.Endpointer[ActivityFunc] = &endpointer{}

type endpointer struct {
	svc activity.Service
}

func (e *endpointer) CallHandlerFunc(f ActivityFunc, ctx context.Context, request any,
	svc any) (platform_http.Errorer, error) {
	return f(ctx, request, svc.(activity.Service)), nil
}

func (e *endpointer) Service() any {
	return e.svc
}

func newUserAuthenticatedEndpointer(fleetSvc fleet.Service, svc activity.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[ActivityFunc] {
	return &eu.CommonEndpointer[ActivityFunc]{
		EP: &endpointer{
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
