package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

const (
	// defaultPerPage is used when per_page is not specified but page is specified.
	defaultPerPage = 20

	// maxPerPage is the maximum allowed value for per_page.
	maxPerPage = 10000
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
	}, parseCustomTags, nil, nil, nil, requestBodySizeLimit)
}

// parseCustomTags handles custom URL tag values for activity requests.
func parseCustomTags(urlTagValue string, r *http.Request, field reflect.Value) (bool, error) {
	if urlTagValue == "http_method" {
		field.Set(reflect.ValueOf(r.Method))
		return true, nil
	}
	return false, nil
}

// handlerFunc is the handler function type for Activity service endpoints.
type handlerFunc func(ctx context.Context, request any, svc api.Service) platform_http.Errorer

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

// Compile-time check to ensure endpointer implements Endpointer.
var _ eu.Endpointer[handlerFunc] = &endpointer{}

func newEndpointerWithAuth(svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
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
