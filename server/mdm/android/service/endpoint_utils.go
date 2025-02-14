package service

import (
	"context"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return endpoint_utils.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response interface{}) error {
			return json.MarshalWrite(w, response, jsontext.WithIndent("  "))
		},
	)
}

func makeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	return endpoint_utils.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.UnmarshalRead(body, req)
	}, nil, nil, nil)
}

// Compile-time check to ensure that authEndpointer implements Endpointer.
var _ endpoint_utils.Endpointer[endpoint_utils.AndroidFunc] = &authEndpointer{}

type authEndpointer struct {
	svc android.Service
}

func (e *authEndpointer) CallHandlerFunc(f endpoint_utils.AndroidFunc, ctx context.Context, request interface{},
	svc interface{}) (fleet.Errorer, error) {
	return f(ctx, request, svc.(android.Service)), nil
}

func (e *authEndpointer) Service() interface{} {
	return e.svc
}

func (e *authEndpointer) StartingAtVersion() string {
	return ""
}

func (e *authEndpointer) SetStartingAtVersion(_ string) {
	panic("not implemented")
}

func (e *authEndpointer) EndingAtVersion() string {
	return ""
}

func (e *authEndpointer) SetEndingAtVersion(_ string) {
	panic("not implemented")
}

func (e *authEndpointer) AlternativePaths() []string {
	return nil
}

func (e *authEndpointer) SetAlternativePaths(_ []string) {
	panic("not implemented")
}

func (e *authEndpointer) UsePathPrefix() bool {
	return false
}

func (e *authEndpointer) SetUsePathPrefix(_ bool) {
	panic("not implemented")
}

func (e *authEndpointer) Copy() endpoint_utils.Endpointer[endpoint_utils.AndroidFunc] {
	result := *e
	return &result
}

func newUserAuthenticatedEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.AndroidFunc] {
	return &endpoint_utils.CommonEndpointer[endpoint_utils.AndroidFunc]{
		EP: &authEndpointer{
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
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.AndroidFunc] {
	return &endpoint_utils.CommonEndpointer[endpoint_utils.AndroidFunc]{
		EP: &authEndpointer{
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
