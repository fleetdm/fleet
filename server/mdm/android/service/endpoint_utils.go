package service

import (
	"context"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	eu "github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return eu.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response interface{}) error {
			return json.MarshalWrite(w, response, jsontext.WithIndent("  "))
		},
	)
}

func makeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.UnmarshalRead(body, req)
	}, nil, nil, nil)
}

// Compile-time check to ensure that endpointer implements Endpointer.
var _ eu.Endpointer[eu.AndroidFunc] = &endpointer{}

type endpointer struct {
	svc android.Service
}

func (e *endpointer) CallHandlerFunc(f eu.AndroidFunc, ctx context.Context, request interface{},
	svc interface{}) (fleet.Errorer, error) {
	return f(ctx, request, svc.(android.Service)), nil
}

func (e *endpointer) Service() interface{} {
	return e.svc
}

func newUserAuthenticatedEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.AndroidFunc] {
	return &eu.CommonEndpointer[eu.AndroidFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      auth.AuthenticatedUser,
		FleetService:  fleetSvc,
		Router:        r,
		Versions:      versions,
	}
}

func newNoAuthEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.AndroidFunc] {
	return &eu.CommonEndpointer[eu.AndroidFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      auth.UnauthenticatedRequest,
		FleetService:  fleetSvc,
		Router:        r,
		Versions:      versions,
	}
}
