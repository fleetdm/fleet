package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/contexts/capabilities"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	eu "github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
)

func makeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, jsonDecode, parseCustomTags, isBodyDecoder, decodeBody)
}

// A value that implements bodyDecoder takes control of decoding the request body.
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

func parseCustomTags(urlTagValue string, r *http.Request, field reflect.Value) (bool, error) {
	switch urlTagValue {
	case "list_options":
		opts, err := listOptionsFromRequest(r)
		if err != nil {
			return false, err
		}
		field.Set(reflect.ValueOf(opts))
		return true, nil

	case "user_options":
		opts, err := userListOptionsFromRequest(r)
		if err != nil {
			return false, err
		}
		field.Set(reflect.ValueOf(opts))
		return true, nil

	case "host_options":
		opts, err := hostListOptionsFromRequest(r)
		if err != nil {
			return false, err
		}
		field.Set(reflect.ValueOf(opts))
		return true, nil

	case "carve_options":
		opts, err := carveListOptionsFromRequest(r)
		if err != nil {
			return false, err
		}
		field.Set(reflect.ValueOf(opts))
		return true, nil
	}
	return false, nil
}

func jsonDecode(body io.Reader, req any) error {
	return json.NewDecoder(body).Decode(req)
}

func isBodyDecoder(v reflect.Value) bool {
	_, ok := v.Interface().(bodyDecoder)
	return ok
}

// Compile-time check to ensure that endpointer implements Endpointer.
var _ eu.Endpointer[eu.HandlerFunc] = &endpointer{}

type endpointer struct {
	svc fleet.Service
}

func (e *endpointer) CallHandlerFunc(f eu.HandlerFunc, ctx context.Context, request interface{},
	svc interface{}) (fleet.Errorer, error) {
	return f(ctx, request, svc.(fleet.Service))
}

func (e *endpointer) Service() interface{} {
	return e.svc
}

func newUserAuthenticatedEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.HandlerFunc] {
	return &eu.CommonEndpointer[eu.HandlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      auth.AuthenticatedUser,
		FleetService:  svc,
		Router:        r,
		Versions:      versions,
	}
}

func newNoAuthEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.HandlerFunc] {
	return &eu.CommonEndpointer[eu.HandlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      auth.UnauthenticatedRequest,
		FleetService:  svc,
		Router:        r,
		Versions:      versions,
	}
}

func badRequest(msg string) error {
	return &fleet.BadRequestError{Message: msg}
}

func newDeviceAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.HandlerFunc] {
	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
		return authenticatedDevice(svc, logger, next)
	}

	// Inject the fleet.CapabilitiesHeader header to the response for device endpoints
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerDeviceCapabilities()))
	// Add the capabilities reported by the device to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &eu.CommonEndpointer[eu.HandlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      authFunc,
		FleetService:  svc,
		Router:        r,
		Versions:      versions,
	}

}

func newHostAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.HandlerFunc] {
	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
		return authenticatedHost(svc, logger, next)
	}
	return &eu.CommonEndpointer[eu.HandlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      authFunc,
		FleetService:  svc,
		Router:        r,
		Versions:      versions,
	}
}

func newOrbitAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[eu.HandlerFunc] {
	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
		return authenticatedOrbitHost(svc, logger, next)
	}

	// Inject the fleet.Capabilities header to the response for Orbit hosts
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerOrbitCapabilities()))
	// Add the capabilities reported by Orbit to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &eu.CommonEndpointer[eu.HandlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthFunc:      authFunc,
		FleetService:  svc,
		Router:        r,
		Versions:      versions,
	}
}

func capabilitiesResponseFunc(capabilities fleet.CapabilityMap) kithttp.ServerOption {
	return kithttp.ServerAfter(func(ctx context.Context, w http.ResponseWriter) context.Context {
		writeCapabilitiesHeader(w, capabilities)
		return ctx
	})
}

func capabilitiesContextFunc() kithttp.ServerOption {
	return kithttp.ServerBefore(capabilities.NewContext)
}

func writeCapabilitiesHeader(w http.ResponseWriter, capabilities fleet.CapabilityMap) {
	if len(capabilities) == 0 {
		return
	}

	w.Header().Set(fleet.CapabilitiesHeader, capabilities.String())
}
