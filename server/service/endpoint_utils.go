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
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/capabilities"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
)

// A value that implements bodyDecoder takes control of decoding the request
// body.
type bodyDecoder interface {
	DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error
}

func MakeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	return endpoint_utils.MakeDecoder(iface, jsonDecode, parseCustomTags, isBodyDecoder, decodeBody)
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

// Compile-time check to ensure that AuthEndpointer implements Endpointer.
var _ endpoint_utils.Endpointer[endpoint_utils.HandlerFunc] = &AuthEndpointer{}

type AuthEndpointer struct {
	svc               fleet.Service
	opts              []kithttp.ServerOption
	r                 *mux.Router
	authFunc          func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint
	versions          []string
	startingAtVersion string
	endingAtVersion   string
	alternativePaths  []string
	customMiddleware  []endpoint.Middleware
	usePathPrefix     bool
}

func (e *AuthEndpointer) CallHandlerFunc(f endpoint_utils.HandlerFunc, ctx context.Context, request interface{},
	svc interface{}) (fleet.Errorer, error) {
	return f(ctx, request, svc.(fleet.Service))
}

func (e *AuthEndpointer) AuthFunc(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
	return e.authFunc(svc, next)
}

func (e *AuthEndpointer) Service() interface{} {
	return e.svc
}

func (e *AuthEndpointer) FleetService() fleet.Service {
	return e.svc
}

func (e *AuthEndpointer) CustomMiddleware() []endpoint.Middleware {
	return e.customMiddleware
}

func (e *AuthEndpointer) SetCustomMiddleware(v []endpoint.Middleware) {
	e.customMiddleware = v
}

func (e *AuthEndpointer) ServerOptions() []kithttp.ServerOption {
	return e.opts
}

func (e *AuthEndpointer) StartingAtVersion() string {
	return e.startingAtVersion
}

func (e *AuthEndpointer) SetStartingAtVersion(v string) {
	e.startingAtVersion = v
}

func (e *AuthEndpointer) EndingAtVersion() string {
	return e.endingAtVersion
}

func (e *AuthEndpointer) SetEndingAtVersion(v string) {
	e.endingAtVersion = v
}

func (e *AuthEndpointer) AlternativePaths() []string {
	return e.alternativePaths
}

func (e *AuthEndpointer) SetAlternativePaths(v []string) {
	e.alternativePaths = v
}

func (e *AuthEndpointer) UsePathPrefix() bool {
	return e.usePathPrefix
}

func (e *AuthEndpointer) SetUsePathPrefix(v bool) {
	e.usePathPrefix = v
}

func (e *AuthEndpointer) Copy() endpoint_utils.Endpointer[endpoint_utils.HandlerFunc] {
	result := *e
	return &result
}

func NewUserAuthenticatedEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc] {
	return &endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc]{
		EP: &AuthEndpointer{
			svc:      svc,
			opts:     opts,
			r:        r,
			authFunc: auth.AuthenticatedUser,
			versions: versions,
		},
		MakeDecoderFn: MakeDecoder,
		EncodeFn:      encodeResponse,
	}
}

func NewNoAuthEndpointer(svc fleet.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc] {
	return &endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc]{
		EP: &AuthEndpointer{
			svc:      svc,
			opts:     opts,
			r:        r,
			authFunc: auth.UnauthenticatedRequest,
			versions: versions,
		},
		MakeDecoderFn: MakeDecoder,
		EncodeFn:      encodeResponse,
	}
}

var pathReplacer = strings.NewReplacer(
	"/", "_",
	"{", "_",
	"}", "_",
)

func getNameFromPathAndVerb(verb, path, startAt string) string {
	prefix := strings.ToLower(verb) + "_"
	if startAt != "" {
		prefix += pathReplacer.Replace(startAt) + "_"
	}
	return prefix + pathReplacer.Replace(strings.TrimPrefix(strings.TrimRight(path, "/"), "/api/_version_/fleet/"))
}

// PathHandler registers a handler for the verb and path. The pathHandler is
// a function that receives the actual path to which it will be mounted, and
// returns the actual http.Handler that will handle this endpoint. This is for
// when the handler needs to know on which path it was called.
func (e *AuthEndpointer) PathHandler(verb, path string, pathHandler func(path string) http.Handler) {
	e.handlePathHandler(path, pathHandler, verb)
}

func (e *AuthEndpointer) handlePathHandler(path string, pathHandler func(path string) http.Handler, verb string) {
	versions := e.versions
	if e.startingAtVersion != "" {
		startIndex := -1
		for i, version := range versions {
			if version == e.startingAtVersion {
				startIndex = i
				break
			}
		}
		if startIndex == -1 {
			panic("StartAtVersion is not part of the valid versions")
		}
		versions = versions[startIndex:]
	}
	if e.endingAtVersion != "" {
		endIndex := -1
		for i, version := range versions {
			if version == e.endingAtVersion {
				endIndex = i
				break
			}
		}
		if endIndex == -1 {
			panic("EndAtVersion is not part of the valid versions")
		}
		versions = versions[:endIndex+1]
	}

	// if a version doesn't have a deprecation version, or the ending version is the latest one, then it's part of the
	// latest
	if e.endingAtVersion == "" || e.endingAtVersion == e.versions[len(e.versions)-1] {
		versions = append(versions, "latest")
	}

	versionedPath := strings.Replace(path, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", strings.Join(versions, "|")), 1)
	nameAndVerb := getNameFromPathAndVerb(verb, path, e.startingAtVersion)
	if e.usePathPrefix {
		e.r.PathPrefix(versionedPath).Handler(pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
	} else {
		e.r.Handle(versionedPath, pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
	}
	for _, alias := range e.alternativePaths {
		nameAndVerb := getNameFromPathAndVerb(verb, alias, e.startingAtVersion)
		versionedPath := strings.Replace(alias, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", strings.Join(versions, "|")), 1)
		if e.usePathPrefix {
			e.r.PathPrefix(versionedPath).Handler(pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
		} else {
			e.r.Handle(versionedPath, pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
		}
	}
}

func (e *AuthEndpointer) HandleHTTPHandler(path string, h http.Handler, verb string) {
	self := func(_ string) http.Handler { return h }
	e.handlePathHandler(path, self, verb)
}

func badRequest(msg string) error {
	return &fleet.BadRequestError{Message: msg}
}

func newDeviceAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc] {
	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
		return authenticatedDevice(svc, logger, next)
	}

	// Inject the fleet.CapabilitiesHeader header to the response for device endpoints
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerDeviceCapabilities()))
	// Add the capabilities reported by the device to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc]{
		EP: &AuthEndpointer{
			svc:      svc,
			opts:     opts,
			r:        r,
			authFunc: authFunc,
			versions: versions,
		},
		MakeDecoderFn: MakeDecoder,
		EncodeFn:      encodeResponse,
	}

}

func newHostAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc] {
	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
		return authenticatedHost(svc, logger, next)
	}
	return &endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc]{
		EP: &AuthEndpointer{
			svc:      svc,
			opts:     opts,
			r:        r,
			authFunc: authFunc,
			versions: versions,
		},
		MakeDecoderFn: MakeDecoder,
		EncodeFn:      encodeResponse,
	}
}

func newOrbitAuthenticatedEndpointer(svc fleet.Service, logger log.Logger, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc] {
	authFunc := func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint {
		return authenticatedOrbitHost(svc, logger, next)
	}

	// Inject the fleet.Capabilities header to the response for Orbit hosts
	opts = append(opts, capabilitiesResponseFunc(fleet.GetServerOrbitCapabilities()))
	// Add the capabilities reported by Orbit to the request context
	opts = append(opts, capabilitiesContextFunc())

	return &endpoint_utils.CommonEndpointer[endpoint_utils.HandlerFunc]{
		EP: &AuthEndpointer{
			svc:      svc,
			opts:     opts,
			r:        r,
			authFunc: authFunc,
			versions: versions,
		},
		MakeDecoderFn: MakeDecoder,
		EncodeFn:      encodeResponse,
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
