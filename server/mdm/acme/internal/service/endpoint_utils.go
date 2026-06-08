package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
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
		acmeDomainErrorEncoder,
	)
}

func acmeErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	var acmeErr *types.ACMEError
	if !errors.As(err, &acmeErr) {
		// TODO: If we can get access to a logger, we can log the details here, to help troubleshoot service errors.
		// if it's not already an ACME error, it is because it is an internal server
		// error (or a dev error, for 4xx we should always return ACMEError).
		acmeErr = types.InternalServerError("") // not passing err.Error() as we don't want to leak internal details
	}

	w.Header().Set("Content-Type", "application/problem+json")
	statusCode := acmeErr.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	// ignoring error as response started being written at that point
	_ = json.NewEncoder(w).Encode(acmeErr)
}

func acmeDomainErrorEncoder(ctx context.Context, err error, w http.ResponseWriter, enc *json.Encoder, jsonErr *eu.JsonError) (handled bool) {
	acmeErrorEncoder(ctx, err, w)
	return true
}

// makeDecoder creates a decoder for the given request type.
func makeDecoder(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.NewDecoder(body).Decode(req)
	}, parseCustomTags, isBodyDecoder, decodeBody, nil, requestBodySizeLimit)
}

// parseCustomTags handles custom URL tag values for acme requests.
func parseCustomTags(urlTagValue string, r *http.Request, field reflect.Value) (bool, error) {
	switch urlTagValue {
	case "http_method":
		field.Set(reflect.ValueOf(r.Method))
		return true, nil
	case "http_path":
		field.Set(reflect.ValueOf(r.URL.Path))
		return true, nil
	}
	return false, nil
}

func isBodyDecoder(v reflect.Value) bool {
	_, ok := v.Interface().(bodyDecoder)
	return ok
}

type bodyDecoder interface {
	DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error
}

func decodeBody(ctx context.Context, r *http.Request, v reflect.Value, body io.Reader) error {
	bd := v.Interface().(bodyDecoder)
	var certs []*x509.Certificate
	if (r.TLS != nil) && (r.TLS.PeerCertificates != nil) {
		certs = r.TLS.PeerCertificates
	}

	if err := bd.DecodeBody(ctx, body, r.URL.Query(), certs); err != nil {
		return err
	}
	return nil
}

// handlerFunc is the handler function type for ACME service endpoints.
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

func newEndpointerWithNoAuth(svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption, r *mux.Router,
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
