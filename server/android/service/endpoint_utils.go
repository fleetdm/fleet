package service

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

type handlerFunc func(ctx context.Context, request interface{}, svc android.Service) (errorer, error)

// parseTag parses a `url` tag and whether it's optional or not, which is an optional part of the tag
func parseTag(tag string) (string, bool, error) {
	parts := strings.Split(tag, ",")
	switch len(parts) {
	case 0:
		return "", false, fmt.Errorf("Error parsing %s: too few parts", tag)
	case 1:
		return tag, false, nil
	case 2:
		return parts[0], parts[1] == "optional", nil
	default:
		return "", false, fmt.Errorf("Error parsing %s: too many parts", tag)
	}
}

type fieldPair struct {
	sf reflect.StructField
	v  reflect.Value
}

// allFields returns all the fields for a struct, including the ones from embedded structs
func allFields(ifv reflect.Value) []fieldPair {
	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
	}
	if ifv.Kind() != reflect.Struct {
		return nil
	}

	var fields []fieldPair

	if !ifv.IsValid() {
		return nil
	}

	t := ifv.Type()

	for i := 0; i < ifv.NumField(); i++ {
		v := ifv.Field(i)

		if v.Kind() == reflect.Struct && t.Field(i).Anonymous {
			fields = append(fields, allFields(v)...)
			continue
		}
		fields = append(fields, fieldPair{sf: ifv.Type().Field(i), v: v})
	}

	return fields
}

// A value that implements requestDecoder takes control of decoding the request
// as a whole - that is, it is responsible for decoding the body and any url
// or query argument itself.
type requestDecoder interface {
	DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error)
}

// A value that implements bodyDecoder takes control of decoding the request
// body.
type bodyDecoder interface {
	DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error
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
	if rd, ok := iface.(requestDecoder); ok {
		return func(ctx context.Context, r *http.Request) (interface{}, error) {
			return rd.DecodeRequest(ctx, r)
		}
	}

	t := reflect.TypeOf(iface)
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("makeDecoder only understands structs, not %T", iface))
	}

	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		v := reflect.New(t)
		nilBody := false

		var isBodyDecoder bool
		if _, ok := v.Interface().(bodyDecoder); ok {
			isBodyDecoder = true
		}

		buf := bufio.NewReader(r.Body)
		var body io.Reader = buf
		if _, err := buf.Peek(1); err == io.EOF {
			nilBody = true
		} else {
			if r.Header.Get("content-encoding") == "gzip" {
				gzr, err := gzip.NewReader(buf)
				if err != nil {
					return nil, badRequestErr("gzip decoder error", err)
				}
				defer gzr.Close()
				body = gzr
			}

			if !isBodyDecoder {
				req := v.Interface()
				if err := json.NewDecoder(body).Decode(req); err != nil {
					return nil, badRequestErr("json decoder error", err)
				}
				v = reflect.ValueOf(req)
			}
		}

		fields := allFields(v)
		for _, fp := range fields {
			field := fp.v

			urlTagValue, ok := fp.sf.Tag.Lookup("url")

			optional := false
			var err error
			if ok {
				urlTagValue, optional, err = parseTag(urlTagValue)
				if err != nil {
					return nil, err
				}
				switch urlTagValue {
				case "list_options":
					opts, err := listOptionsFromRequest(r)
					if err != nil {
						return nil, err
					}
					field.Set(reflect.ValueOf(opts))

				case "user_options":
					opts, err := userListOptionsFromRequest(r)
					if err != nil {
						return nil, err
					}
					field.Set(reflect.ValueOf(opts))

				case "host_options":
					opts, err := hostListOptionsFromRequest(r)
					if err != nil {
						return nil, err
					}
					field.Set(reflect.ValueOf(opts))

				case "carve_options":
					opts, err := carveListOptionsFromRequest(r)
					if err != nil {
						return nil, err
					}
					field.Set(reflect.ValueOf(opts))

				default:
					switch field.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						v, err := intFromRequest(r, urlTagValue)
						if err != nil {
							if err == errBadRoute && optional {
								continue
							}
							return nil, badRequestErr("intFromRequest", err)
						}
						field.SetInt(v)

					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						v, err := uintFromRequest(r, urlTagValue)
						if err != nil {
							if err == errBadRoute && optional {
								continue
							}
							return nil, badRequestErr("uintFromRequest", err)
						}
						field.SetUint(v)

					case reflect.String:
						v, err := stringFromRequest(r, urlTagValue)
						if err != nil {
							if err == errBadRoute && optional {
								continue
							}
							return nil, badRequestErr("stringFromRequest", err)
						}
						field.SetString(v)

					default:
						return nil, fmt.Errorf("unsupported type for field %s for 'url' decoding: %s", urlTagValue, field.Kind())
					}
				}
			}

			_, jsonExpected := fp.sf.Tag.Lookup("json")
			if jsonExpected && nilBody {
				return nil, badRequest("Expected JSON Body")
			}

			queryTagValue, ok := fp.sf.Tag.Lookup("query")

			if ok {
				queryTagValue, optional, err = parseTag(queryTagValue)
				if err != nil {
					return nil, err
				}
				queryVal := r.URL.Query().Get(queryTagValue)
				// if optional and it's a ptr, leave as nil
				if queryVal == "" {
					if optional {
						continue
					}
					return nil, badRequest(fmt.Sprintf("Param %s is required", fp.sf.Name))
				}
				if field.Kind() == reflect.Ptr {
					// create the new instance of whatever it is
					field.Set(reflect.New(field.Type().Elem()))
					field = field.Elem()
				}
				switch field.Kind() {
				case reflect.String:
					field.SetString(queryVal)
				case reflect.Uint:
					queryValUint, err := strconv.Atoi(queryVal)
					if err != nil {
						return nil, badRequestErr("parsing uint from query", err)
					}
					field.SetUint(uint64(queryValUint)) //nolint:gosec // dismiss G115
				case reflect.Float64:
					queryValFloat, err := strconv.ParseFloat(queryVal, 64)
					if err != nil {
						return nil, badRequestErr("parsing float from query", err)
					}
					field.SetFloat(queryValFloat)
				case reflect.Bool:
					field.SetBool(queryVal == "1" || queryVal == "true")
				case reflect.Int:
					queryValInt := 0
					switch queryTagValue {
					case "order_direction", "inherited_order_direction":
						switch queryVal {
						case "desc":
							queryValInt = int(fleet.OrderDescending)
						case "asc":
							queryValInt = int(fleet.OrderAscending)
						case "":
							queryValInt = int(fleet.OrderAscending)
						default:
							return fleet.ListOptions{}, badRequest("unknown order_direction: " + queryVal)
						}
					default:
						queryValInt, err = strconv.Atoi(queryVal)
						if err != nil {
							return nil, badRequestErr("parsing int from query", err)
						}
					}
					field.SetInt(int64(queryValInt))
				default:
					return nil, fmt.Errorf("Cant handle type for field %s %s", fp.sf.Name, field.Kind())
				}
			}
		}

		if isBodyDecoder {
			bd := v.Interface().(bodyDecoder)
			var certs []*x509.Certificate
			if (r.TLS != nil) && (r.TLS.PeerCertificates != nil) {
				certs = r.TLS.PeerCertificates
			}

			if err := bd.DecodeBody(ctx, body, r.URL.Query(), certs); err != nil {
				return nil, err
			}
		}

		if !license.IsPremium(ctx) {
			for _, fp := range fields {
				if prem, ok := fp.sf.Tag.Lookup("premium"); ok {
					val, err := strconv.ParseBool(prem)
					if err != nil {
						return nil, err
					}
					if val && !fp.v.IsZero() {
						return nil, &fleet.BadRequestError{Message: fmt.Sprintf(
							"option %s requires a premium license",
							fp.sf.Name,
						)}
					}
					continue
				}
			}
		}

		return v.Interface(), nil
	}
}

func badRequest(msg string) error {
	return &fleet.BadRequestError{Message: msg}
}

func badRequestErr(publicMsg string, internalErr error) error {
	// ensure timeout errors don't become BadRequestErrors.
	var opErr *net.OpError
	if errors.As(internalErr, &opErr) {
		return fmt.Errorf(publicMsg+", internal: %w", internalErr)
	}
	return &fleet.BadRequestError{
		Message:     publicMsg,
		InternalErr: internalErr,
	}
}

type authEndpointer struct {
	fleetSvc          fleet.Service
	svc               android.Service
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

func newUserAuthenticatedEndpointer(fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *authEndpointer {
	return &authEndpointer{
		fleetSvc: fleetSvc,
		svc:      svc,
		opts:     opts,
		r:        r,
		authFunc: authenticatedUser,
		versions: versions,
	}
}

func newNoAuthEndpointer(svc android.Service, opts []kithttp.ServerOption, r *mux.Router, versions ...string) *authEndpointer {
	return &authEndpointer{
		fleetSvc: nil,
		svc:      svc,
		opts:     opts,
		r:        r,
		authFunc: unauthenticatedRequest,
		versions: versions,
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

func writeBrowserSecurityHeaders(w http.ResponseWriter) {
	// Strict-Transport-Security informs browsers that the site should only be
	// accessed using HTTPS, and that any future attempts to access it using
	// HTTP should automatically be converted to HTTPS.
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains;")
	// X-Frames-Options disallows embedding the UI in other sites via <frame>,
	// <iframe>, <embed> or <object>, which can prevent attacks like
	// clickjacking.
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	// X-Content-Type-Options prevents browsers from trying to guess the MIME
	// type which can cause browsers to transform non-executable content into
	// executable content.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Referrer-Policy prevents leaking the origin of the referrer in the
	// Referer.
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
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

// PathHandler registers a handler for the verb and path. The pathHandler is
// a function that receives the actual path to which it will be mounted, and
// returns the actual http.Handler that will handle this endpoint. This is for
// when the handler needs to know on which path it was called.
func (e *authEndpointer) PathHandler(verb, path string, pathHandler func(path string) http.Handler) {
	e.handlePathHandler(path, pathHandler, verb)
}

func (e *authEndpointer) handlePathHandler(path string, pathHandler func(path string) http.Handler, verb string) {
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

func (e *authEndpointer) handleHTTPHandler(path string, h http.Handler, verb string) {
	self := func(_ string) http.Handler { return h }
	e.handlePathHandler(path, self, verb)
}

func (e *authEndpointer) handleEndpoint(path string, f handlerFunc, v interface{}, verb string) {
	endpoint := e.makeEndpoint(f, v)
	e.handleHTTPHandler(path, endpoint, verb)
}

func (e *authEndpointer) makeEndpoint(f handlerFunc, v interface{}) http.Handler {
	next := func(ctx context.Context, request interface{}) (interface{}, error) {
		return f(ctx, request, e.svc)
	}
	endp := e.authFunc(e.fleetSvc, next)

	// apply middleware in reverse order so that the first wraps the second
	// wraps the third etc.
	for i := len(e.customMiddleware) - 1; i >= 0; i-- {
		mw := e.customMiddleware[i]
		endp = mw(endp)
	}

	return newServer(endp, makeDecoder(v), e.opts)
}

func (e *authEndpointer) StartingAtVersion(version string) *authEndpointer {
	ae := *e
	ae.startingAtVersion = version
	return &ae
}

func (e *authEndpointer) EndingAtVersion(version string) *authEndpointer {
	ae := *e
	ae.endingAtVersion = version
	return &ae
}

func (e *authEndpointer) WithAltPaths(paths ...string) *authEndpointer {
	ae := *e
	ae.alternativePaths = paths
	return &ae
}

func (e *authEndpointer) WithCustomMiddleware(mws ...endpoint.Middleware) *authEndpointer {
	ae := *e
	ae.customMiddleware = mws
	return &ae
}

func (e *authEndpointer) UsePathPrefix() *authEndpointer {
	ae := *e
	ae.usePathPrefix = true
	return &ae
}
