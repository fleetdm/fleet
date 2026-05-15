package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/chart/api"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// encodeResponse encodes the response as JSON using the common Fleet encoding pattern.
func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	return eu.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response any) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(response)
		},
		nil, // no domain-specific error encoder; standard fleet errors are handled by common encoder
	)
}

// makeDecoder creates a decoder for the given request type.
func makeDecoder(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.NewDecoder(body).Decode(req)
	}, nil, nil, nil, nil, requestBodySizeLimit)
}

// handlerFunc is the handler function type for chart service endpoints.
type handlerFunc func(ctx context.Context, request any, svc api.Service) (platform_http.Errorer, error)

type chartEndpointer struct {
	svc api.Service
}

func (e *chartEndpointer) CallHandlerFunc(f handlerFunc, ctx context.Context, request any, svc any) (platform_http.Errorer, error) {
	return f(ctx, request, svc.(api.Service))
}

func (e *chartEndpointer) Service() any {
	return e.svc
}

// Compile-time check to ensure chartEndpointer implements Endpointer.
var _ eu.Endpointer[handlerFunc] = &chartEndpointer{}

func newChartEndpointer(svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &chartEndpointer{
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
