package service

// TODO(26218): Refactor this to remove duplication.

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

type handlerFunc func(ctx context.Context, request interface{}, svc android.Service) errorer

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.Error() != nil {
		endpoint_utils.EncodeError(ctx, e.Error(), w)
		return nil
	}

	if e, ok := response.(statuser); ok {
		w.WriteHeader(e.Status())
		if e.Status() == http.StatusNoContent {
			return nil
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

// statuser allows response types to implement a custom
// http success status - default is 200 OK
type statuser interface {
	Status() int
}

// errorer interface is implemented by response structs to encode business logic errors
type errorer interface {
	Error() error
}

// makeDecoder creates a decoder for the type for the struct passed on. If the
// struct has at least 1 json tag it'll unmarshall the body. If the struct has
// a `url` tag with value list_options it'll gather fleet.ListOptions from the
// URL (similarly for host_options, carve_options, user_options that derive
// from the common list_options). Note that these behaviors do not work for embedded structs.
//
// Finally, any other `url` tag will be treated as a path variable (of the form
// /path/{name} in the route's path) from the URL path pattern, and it'll be
// decoded and set accordingly. Variables can be optional by setting the tag as
// follows: `url:"some-id,optional"`.
// The "list_options" are optional by default and it'll ignore the optional
// portion of the tag.
//
// If iface implements the requestDecoder interface, it returns a function that
// calls iface.DecodeRequest(ctx, r) - i.e. the value itself fully controls its
// own decoding.
//
// If iface implements the bodyDecoder interface, it calls iface.DecodeBody
// after having decoded any non-body fields (such as url and query parameters)
// into the struct.
func makeDecoder(iface interface{}) kithttp.DecodeRequestFunc {
	if iface == nil {
		return func(ctx context.Context, r *http.Request) (interface{}, error) {
			return nil, nil
		}
	}

	t := reflect.TypeOf(iface)
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("makeDecoder only understands structs, not %T", iface))
	}

	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		v := reflect.New(t)
		nilBody := false
		buf := bufio.NewReader(r.Body)
		var body io.Reader = buf
		if _, err := buf.Peek(1); err == io.EOF {
			nilBody = true
		} else {
			if r.Header.Get("content-encoding") == "gzip" {
				gzr, err := gzip.NewReader(buf)
				if err != nil {
					return nil, endpoint_utils.BadRequestErr("gzip decoder error", err)
				}
				defer gzr.Close()
				body = gzr
			}

			req := v.Interface()
			if err := json.NewDecoder(body).Decode(req); err != nil {
				return nil, endpoint_utils.BadRequestErr("json decoder error", err)
			}
			v = reflect.ValueOf(req)
		}

		fields := endpoint_utils.AllFields(v)
		for _, fp := range fields {
			field := fp.V

			urlTagValue, ok := fp.Sf.Tag.Lookup("url")

			var err error
			if ok {
				optional := false
				urlTagValue, optional, err = endpoint_utils.ParseTag(urlTagValue)
				if err != nil {
					return nil, err
				}
				err = endpoint_utils.DecodeURLTagValue(r, field, urlTagValue, optional)
				if err != nil {
					return nil, err
				}
				continue
			}

			_, jsonExpected := fp.Sf.Tag.Lookup("json")
			if jsonExpected && nilBody {
				return nil, badRequest("Expected JSON Body")
			}

			err = endpoint_utils.DecodeQueryTagValue(r, fp)
			if err != nil {
				return nil, err
			}
		}

		return v.Interface(), nil
	}
}

func badRequest(msg string) error {
	return &fleet.BadRequestError{Message: msg}
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
