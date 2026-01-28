package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/contexts/capabilities"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
)

func makeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, jsonDecode, parseCustomTags, isBodyDecoder, decodeBody, fleetQueryDecoder)
}

// fleetQueryDecoder handles fleet-specific query parameter decoding, such as
// converting the order_direction string to the fleet.OrderDirection int type.
func fleetQueryDecoder(queryTagName, queryVal string, field reflect.Value) (bool, error) {
	// Only handle int fields for order_direction
	if field.Kind() != reflect.Int {
		return false, nil
	}
	switch queryTagName {
	case "order_direction", "inherited_order_direction":
		var direction int
		switch queryVal {
		case "desc":
			direction = int(fleet.OrderDescending)
		case "asc":
			direction = int(fleet.OrderAscending)
		default:
			return false, &fleet.BadRequestError{Message: "unknown order_direction: " + queryVal}
		}
		field.SetInt(int64(direction))
		return true, nil
	}
	return false, nil
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

// handlerFunc is the handler function type for the main Fleet service endpoints.
type handlerFunc func(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error)

// Compile-time check to ensure that fleetEndpointer implements Endpointer.
var _ eu.Endpointer[handlerFunc] = &fleetEndpointer{}

type fleetEndpointer struct {
	svc fleet.Service
}

func (e *fleetEndpointer) CallHandlerFunc(f handlerFunc, ctx context.Context, request any, svc any) (platform_http.Errorer, error) {
	return f(ctx, request, svc.(fleet.Service))
}

func (e *fleetEndpointer) Service() any {
	return e.svc
}

func newUserAuthenticatedEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return auth.AuthenticatedUser(svc, next)
		},
		Router:   r,
		Versions: versions,
	}
}

func newNoAuthEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return auth.UnauthenticatedRequest(svc, next)
		},
		Router:   r,
		Versions: versions,
	}
}

func newOrbitNoAuthEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	// Add the capabilities reported by Orbit to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return auth.UnauthenticatedRequest(svc, next)
		},
		Router:   r,
		Versions: versions,
	}
}

func badRequest(msg string) error {
	return &fleet.BadRequestError{Message: msg}
}

func badRequestf(format string, a ...any) error {
	return &fleet.BadRequestError{
		Message: fmt.Sprintf(format, a...),
	}
}

func newDeviceAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	// Extract certificate serial from X-Client-Cert-Serial header for certificate-based auth
	opts = append(opts, kithttp.ServerBefore(extractCertSerialFromHeader))
	// Inject the fleet.CapabilitiesHeader header to the response for device endpoints
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerDeviceCapabilities()))
	// Add the capabilities reported by the device to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return authenticatedDevice(svc, logger, next)
		},
		Router:   r,
		Versions: versions,
	}
}

func newHostAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return authenticatedHost(svc, logger, next)
		},
		Router:   r,
		Versions: versions,
	}
}

func androidAuthenticatedEndpointer(
	svc fleet.Service,
	logger log.Logger,
	opts []kithttp.ServerOption,
	r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	// Inject the fleet.Capabilities header to the response for Orbit hosts
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerOrbitCapabilities()))
	// Add the capabilities reported by Orbit to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return authenticatedOrbitHost(svc, logger, next, authHeaderValue("Node key "))
		},
		Router:   r,
		Versions: versions,
	}
}

func newOrbitAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string,
) *eu.CommonEndpointer[handlerFunc] {
	// Inject the fleet.Capabilities header to the response for Orbit hosts
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerOrbitCapabilities()))
	// Add the capabilities reported by Orbit to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &eu.CommonEndpointer[handlerFunc]{
		EP: &fleetEndpointer{
			svc: svc,
		},
		MakeDecoderFn: makeDecoder,
		EncodeFn:      encodeResponse,
		Opts:          opts,
		AuthMiddleware: func(next endpoint.Endpoint) endpoint.Endpoint {
			return authenticatedOrbitHost(svc, logger, next, getOrbitNodeKey)
		},
		Router:   r,
		Versions: versions,
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
