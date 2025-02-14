package service

// TODO(26218): Refactor this to remove duplication.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

type handlerFunc func(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer

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

type authEndpointer struct {
	fleetSvc         fleet.Service
	svc              android.Service
	opts             []kithttp.ServerOption
	r                *mux.Router
	authFunc         func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint
	versions         []string
	customMiddleware []endpoint.Middleware
}

func newUserAuthenticatedEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *authEndpointer {
	return &authEndpointer{
		fleetSvc: fleetSvc,
		svc:      svc,
		opts:     opts,
		r:        r,
		authFunc: auth.AuthenticatedUser,
		versions: versions,
	}
}

func newNoAuthEndpointer(svc android.Service, opts []kithttp.ServerOption, r *mux.Router, versions ...string) *authEndpointer {
	return &authEndpointer{
		fleetSvc: nil,
		svc:      svc,
		opts:     opts,
		r:        r,
		authFunc: auth.UnauthenticatedRequest,
		versions: versions,
	}
}

var pathReplacer = strings.NewReplacer(
	"/", "_",
	"{", "_",
	"}", "_",
)

func getNameFromPathAndVerb(verb, path string) string {
	prefix := strings.ToLower(verb) + "_"
	return prefix + pathReplacer.Replace(strings.TrimPrefix(strings.TrimRight(path, "/"), "/api/_version_/fleet/"))
}

func (e *authEndpointer) POST(path string, f handlerFunc, v interface{}) {
	e.handleEndpoint(path, f, v, "POST")
}

func (e *authEndpointer) GET(path string, f handlerFunc, v interface{}) {
	e.handleEndpoint(path, f, v, "GET")
}

func (e *authEndpointer) PUT(path string, f handlerFunc, v interface{}) {
	e.handleEndpoint(path, f, v, "PUT")
}

func (e *authEndpointer) PATCH(path string, f handlerFunc, v interface{}) {
	e.handleEndpoint(path, f, v, "PATCH")
}

func (e *authEndpointer) DELETE(path string, f handlerFunc, v interface{}) {
	e.handleEndpoint(path, f, v, "DELETE")
}

func (e *authEndpointer) HEAD(path string, f handlerFunc, v interface{}) {
	e.handleEndpoint(path, f, v, "HEAD")
}

func (e *authEndpointer) handlePathHandler(path string, pathHandler func(path string) http.Handler, verb string) {
	versions := e.versions
	versionedPath := strings.Replace(path, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", strings.Join(versions, "|")), 1)
	nameAndVerb := getNameFromPathAndVerb(verb, path)
	e.r.Handle(versionedPath, pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
}

func (e *authEndpointer) handleHTTPHandler(path string, h http.Handler, verb string) {
	self := func(_ string) http.Handler { return h }
	e.handlePathHandler(path, self, verb)
}

func (e *authEndpointer) handleEndpoint(path string, f handlerFunc, v interface{}, verb string) {
	e.handleHTTPHandler(path, e.makeEndpoint(f, v), verb)
}

func (e *authEndpointer) makeEndpoint(f handlerFunc, v interface{}) http.Handler {
	next := func(ctx context.Context, request interface{}) (interface{}, error) {
		return f(ctx, request, e.svc), nil
	}
	endPt := e.authFunc(e.fleetSvc, next)

	// apply middleware in reverse order so that the first wraps the second
	// wraps the third etc.
	for i := len(e.customMiddleware) - 1; i >= 0; i-- {
		mw := e.customMiddleware[i]
		endPt = mw(endPt)
	}

	return newServer(endPt, makeDecoder(v), e.opts)
}
