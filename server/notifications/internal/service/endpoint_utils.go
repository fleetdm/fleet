package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// encodeResponse encodes the response as JSON.
func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	return eu.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response any) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(response)
		},
		nil, // no domain-specific error encoder
	)
}

// makeDecoder creates a decoder for the given request type.
func makeDecoder(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.NewDecoder(body).Decode(req)
	}, nil, nil, nil, nil, requestBodySizeLimit)
}

// handlerFunc is the handler function type for notifications service endpoints.
type handlerFunc func(ctx context.Context, request any, svc api.Service) platform_http.Errorer

// Compile-time check to ensure endpointer implements Endpointer.
var _ eu.Endpointer[handlerFunc] = &endpointer{}

type endpointer struct {
	svc api.Service
}

func (e *endpointer) CallHandlerFunc(f handlerFunc, ctx context.Context,
	request any,
	svc any,
) (platform_http.Errorer, error) {
	return f(ctx, request, svc.(api.Service)), nil
}

func (e *endpointer) Service() any {
	return e.svc
}

// newDeviceAuthenticatedEndpointer builds an endpointer for endpoints
// authenticated by a device auth token. authMiddleware is built by
// server/service (which owns device authentication) and injected here,
// translating the authenticated host down to just its ID before this
// context ever sees it.
func newDeviceAuthenticatedEndpointer(svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	opts = append(opts[:len(opts):len(opts)], kithttp.ServerBefore(eu.RouteTemplateRequestFunc))
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn:  makeDecoder,
		EncodeFn:       encodeResponse,
		Opts:           opts,
		AuthMiddleware: authMiddleware,
		Router:         r,
		Versions:       versions,
	}
}
