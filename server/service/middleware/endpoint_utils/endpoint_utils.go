package endpoint_utils

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck"
	"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
)

type HandlerRoutesFunc func(r *mux.Router, opts []kithttp.ServerOption)

// ParseTag parses a `url` tag and whether it's optional or not, which is an optional part of the tag
func ParseTag(tag string) (string, bool, error) {
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
	Sf reflect.StructField
	V  reflect.Value
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
		fields = append(fields, fieldPair{Sf: ifv.Type().Field(i), V: v})
	}

	return fields
}

func BadRequestErr(publicMsg string, internalErr error) error {
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

func UintFromRequest(r *http.Request, name string) (uint64, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return 0, ErrBadRoute
	}
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, ctxerr.Wrap(r.Context(), err, "UintFromRequest")
	}
	return u, nil
}

func IntFromRequest(r *http.Request, name string) (int64, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return 0, ErrBadRoute
	}
	u, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ctxerr.Wrap(r.Context(), err, "IntFromRequest")
	}
	return u, nil
}

func StringFromRequest(r *http.Request, name string) (string, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return "", ErrBadRoute
	}
	unescaped, err := url.PathUnescape(s)
	if err != nil {
		return "", ctxerr.Wrap(r.Context(), err, "unescape value in path")
	}
	return unescaped, nil
}

func DecodeURLTagValue(r *http.Request, field reflect.Value, urlTagValue string, optional bool) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := IntFromRequest(r, urlTagValue)
		if err != nil {
			if errors.Is(err, ErrBadRoute) && optional {
				return nil
			}
			return BadRequestErr("IntFromRequest", err)
		}
		field.SetInt(v)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := UintFromRequest(r, urlTagValue)
		if err != nil {
			if errors.Is(err, ErrBadRoute) && optional {
				return nil
			}
			return BadRequestErr("UintFromRequest", err)
		}
		field.SetUint(v)

	case reflect.String:
		v, err := StringFromRequest(r, urlTagValue)
		if err != nil {
			if errors.Is(err, ErrBadRoute) && optional {
				return nil
			}
			return BadRequestErr("StringFromRequest", err)
		}
		field.SetString(v)

	default:
		return fmt.Errorf("unsupported type for field %s for 'url' decoding: %s", urlTagValue, field.Kind())
	}
	return nil
}

func DecodeQueryTagValue(r *http.Request, fp fieldPair) error {
	queryTagValue, ok := fp.Sf.Tag.Lookup("query")

	if ok {
		var err error
		var optional bool
		queryTagValue, optional, err = ParseTag(queryTagValue)
		if err != nil {
			return err
		}
		queryVal := r.URL.Query().Get(queryTagValue)
		// if optional and it's a ptr, leave as nil
		if queryVal == "" {
			if optional {
				return nil
			}
			return &fleet.BadRequestError{Message: fmt.Sprintf("Param %s is required", fp.Sf.Name)}
		}
		field := fp.V
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
				return BadRequestErr("parsing uint from query", err)
			}
			field.SetUint(uint64(queryValUint)) //nolint:gosec // dismiss G115
		case reflect.Float64:
			queryValFloat, err := strconv.ParseFloat(queryVal, 64)
			if err != nil {
				return BadRequestErr("parsing float from query", err)
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
				default:
					return &fleet.BadRequestError{Message: "unknown order_direction: " + queryVal}
				}
			default:
				queryValInt, err = strconv.Atoi(queryVal)
				if err != nil {
					return BadRequestErr("parsing int from query", err)
				}
			}
			field.SetInt(int64(queryValInt))
		default:
			return fmt.Errorf("Cant handle type for field %s %s", fp.Sf.Name, field.Kind())
		}
	}
	return nil
}

// copied from https://github.com/go-chi/chi/blob/c97bc988430d623a14f50b7019fb40529036a35a/middleware/realip.go#L42
var (
	trueClientIP  = http.CanonicalHeaderKey("True-Client-IP")
	xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
)

func ExtractIP(r *http.Request) string {
	ip := r.RemoteAddr
	if i := strings.LastIndexByte(ip, ':'); i != -1 {
		ip = ip[:i]
	}

	// Prefer True-Client-IP and X-Real-IP headers before X-Forwarded-For:
	// - True-Client-IP: set by some CDNs (e.g., Akamai) to indicate the real client IP early in the chain
	// - X-Real-IP: set by Nginx or similar proxies as a simpler alternative to X-Forwarded-For
	// These headers are less likely to be spoofed or malformed compared to X-Forwarded-For.
	if tcip := r.Header.Get(trueClientIP); tcip != "" {
		ip = tcip
	} else if xrip := r.Header.Get(xRealIP); xrip != "" {
		ip = xrip
	} else if xff := r.Header.Get(xForwardedFor); xff != "" {
		// X-Forwarded-For is a comma-separated list of IP addresses representing the chain of proxies
		// that a request has passed through. This is not a standard, but a convention.
		// The convention is to treat the left-most IP address as the original client IP.
		// For example:
		//     X-Forwarded-For: 198.51.100.1, 203.0.113.5, 127.0.0.1
		// Means:
		//     - 198.51.100.1 is the client IP
		//     - 127.0.0.1 is the last proxy (likely this server or a local proxy)
		//
		// If the left-most IP is a private or loopback address (e.g., 127.0.0.1 or 10.x.x.x), it may indicate:
		//   - The request originated from a local proxy, or
		//   - The header was spoofed by a client (untrusted source)
		//
		// Having multiple X-Forwarded-For headers is non-standard, so we do not handle it here.
		//
		// Here, we grab the left-most IP address by convention.
		i := strings.Index(xff, ",")
		if i == -1 {
			i = len(xff)
		}
		ip = xff[:i]
	}

	return ip
}

type ErrorHandler struct {
	Logger log.Logger
}

func (h *ErrorHandler) Handle(ctx context.Context, err error) {
	// get the request path
	path, _ := ctx.Value(kithttp.ContextKeyRequestPath).(string)
	logger := level.Info(log.With(h.Logger, "path", path))

	if startTime, ok := logging.StartTime(ctx); ok && !startTime.IsZero() {
		logger = log.With(logger, "took", time.Since(startTime))
	}

	var ewi fleet.ErrWithInternal
	if errors.As(err, &ewi) {
		logger = log.With(logger, "internal", ewi.Internal())
	}

	var ewlf fleet.ErrWithLogFields
	if errors.As(err, &ewlf) {
		logger = log.With(logger, ewlf.LogFields()...)
	}

	var uuider fleet.ErrorUUIDer
	if errors.As(err, &uuider) {
		logger = log.With(logger, "uuid", uuider.UUID())
	}

	var rle ratelimit.Error
	if errors.As(err, &rle) {
		res := rle.Result()
		logger.Log("err", "limit exceeded", "retry_after", res.RetryAfter)
	} else {
		logger.Log("err", err)
	}
}

// A value that implements RequestDecoder takes control of decoding the request
// as a whole - that is, it is responsible for decoding the body and any url
// or query argument itself.
type RequestDecoder interface {
	DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error)
}

// A value that implements requestValidator is called after having the values
// decoded into it to apply further validations.
type requestValidator interface {
	ValidateRequest() error
}

// MakeDecoder creates a decoder for the type for the struct passed on. If the
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
// If iface implements the RequestDecoder interface, it returns a function that
// calls iface.DecodeRequest(ctx, r) - i.e. the value itself fully controls its
// own decoding.
//
// If iface implements the bodyDecoder interface, it calls iface.DecodeBody
// after having decoded any non-body fields (such as url and query parameters)
// into the struct.
func MakeDecoder(
	iface interface{},
	jsonUnmarshal func(body io.Reader, req any) error,
	parseCustomTags func(urlTagValue string, r *http.Request, field reflect.Value) (bool, error),
	isBodyDecoder func(reflect.Value) bool,
	decodeBody func(ctx context.Context, r *http.Request, v reflect.Value, body io.Reader) error,
) kithttp.DecodeRequestFunc {
	if iface == nil {
		return func(ctx context.Context, r *http.Request) (interface{}, error) {
			return nil, nil
		}
	}
	if rd, ok := iface.(RequestDecoder); ok {
		return func(ctx context.Context, r *http.Request) (interface{}, error) {
			return rd.DecodeRequest(ctx, r)
		}
	}

	t := reflect.TypeOf(iface)
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("MakeDecoder only understands structs, not %T", iface))
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
					return nil, BadRequestErr("gzip decoder error", err)
				}
				defer gzr.Close()
				body = gzr
			}

			if isBodyDecoder == nil || !isBodyDecoder(v) {
				req := v.Interface()
				err := jsonUnmarshal(body, req)
				if err != nil {
					return nil, BadRequestErr("json decoder error", err)
				}
				v = reflect.ValueOf(req)
			}
		}

		fields := allFields(v)
		for _, fp := range fields {
			field := fp.V

			urlTagValue, ok := fp.Sf.Tag.Lookup("url")

			var err error
			if ok {
				optional := false
				urlTagValue, optional, err = ParseTag(urlTagValue)
				if err != nil {
					return nil, err
				}
				foundValue := false
				if parseCustomTags != nil {
					foundValue, err = parseCustomTags(urlTagValue, r, field)
					if err != nil {
						return nil, err
					}
				}

				if !foundValue {
					err := DecodeURLTagValue(r, field, urlTagValue, optional)
					if err != nil {
						return nil, err
					}
					continue
				}

			}

			_, jsonExpected := fp.Sf.Tag.Lookup("json")
			if jsonExpected && nilBody {
				return nil, &fleet.BadRequestError{Message: "Expected JSON Body"}
			}

			isContentJson := r.Header.Get("Content-Type") == "application/json"
			isCrossSite := r.Header.Get("Origin") != "" || r.Header.Get("Referer") != ""
			if jsonExpected && isCrossSite && !isContentJson {
				return nil, fleet.NewUserMessageError(errors.New("Expected Content-Type \"application/json\""), http.StatusUnsupportedMediaType)
			}

			err = DecodeQueryTagValue(r, fp)
			if err != nil {
				return nil, err
			}
		}

		if isBodyDecoder != nil && isBodyDecoder(v) {
			err := decodeBody(ctx, r, v, body)
			if err != nil {
				return nil, err
			}
		}

		if !license.IsPremium(ctx) {
			for _, fp := range fields {
				if prem, ok := fp.Sf.Tag.Lookup("premium"); ok {
					val, err := strconv.ParseBool(prem)
					if err != nil {
						return nil, err
					}
					if val && !fp.V.IsZero() {
						return nil, &fleet.BadRequestError{Message: fmt.Sprintf(
							"option %s requires a premium license",
							fp.Sf.Name,
						)}
					}
					continue
				}
			}
		}

		if rv, ok := v.Interface().(requestValidator); ok {
			if err := rv.ValidateRequest(); err != nil {
				return nil, err
			}
		}
		return v.Interface(), nil
	}
}

func WriteBrowserSecurityHeaders(w http.ResponseWriter) {
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

type HandlerFunc func(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error)

type AndroidFunc func(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer

type CommonEndpointer[H HandlerFunc | AndroidFunc] struct {
	EP               Endpointer[H]
	MakeDecoderFn    func(iface interface{}) kithttp.DecodeRequestFunc
	EncodeFn         kithttp.EncodeResponseFunc
	Opts             []kithttp.ServerOption
	AuthFunc         func(svc fleet.Service, next endpoint.Endpoint) endpoint.Endpoint
	FleetService     fleet.Service
	Router           *mux.Router
	CustomMiddleware []endpoint.Middleware
	Versions         []string

	startingAtVersion string
	endingAtVersion   string
	alternativePaths  []string
	usePathPrefix     bool
}

type Endpointer[H HandlerFunc | AndroidFunc] interface {
	CallHandlerFunc(f H, ctx context.Context, request interface{}, svc interface{}) (fleet.Errorer, error)
	Service() interface{}
}

func (e *CommonEndpointer[H]) POST(path string, f H, v interface{}) {
	e.handleEndpoint(path, f, v, "POST")
}

func (e *CommonEndpointer[H]) GET(path string, f H, v interface{}) {
	e.handleEndpoint(path, f, v, "GET")
}

func (e *CommonEndpointer[H]) PUT(path string, f H, v interface{}) {
	e.handleEndpoint(path, f, v, "PUT")
}

func (e *CommonEndpointer[H]) PATCH(path string, f H, v interface{}) {
	e.handleEndpoint(path, f, v, "PATCH")
}

func (e *CommonEndpointer[H]) DELETE(path string, f H, v interface{}) {
	e.handleEndpoint(path, f, v, "DELETE")
}

func (e *CommonEndpointer[H]) HEAD(path string, f H, v interface{}) {
	e.handleEndpoint(path, f, v, "HEAD")
}

func (e *CommonEndpointer[H]) handleEndpoint(path string, f H, v interface{}, verb string) {
	endpoint := e.makeEndpoint(f, v)
	e.HandleHTTPHandler(path, endpoint, verb)
}

func (e *CommonEndpointer[H]) makeEndpoint(f H, v interface{}) http.Handler {
	next := func(ctx context.Context, request interface{}) (interface{}, error) {
		return e.EP.CallHandlerFunc(f, ctx, request, e.EP.Service())
	}
	endp := e.AuthFunc(e.FleetService, next)

	// apply middleware in reverse order so that the first wraps the second
	// wraps the third etc.
	for i := len(e.CustomMiddleware) - 1; i >= 0; i-- {
		mw := e.CustomMiddleware[i]
		endp = mw(endp)
	}

	return newServer(endp, e.MakeDecoderFn(v), e.EncodeFn, e.Opts)
}

func newServer(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc, encodeFn kithttp.EncodeResponseFunc,
	opts []kithttp.ServerOption,
) http.Handler {
	// TODO: some handlers don't have authz checks, and because the SkipAuth call is done only in the
	// endpoint handler, any middleware that raises errors before the handler is reached will end up
	// returning authz check missing instead of the more relevant error. Should be addressed as part
	// of #4406.
	e = authzcheck.NewMiddleware().AuthzCheck()(e)
	return kithttp.NewServer(e, decodeFn, encodeFn, opts...)
}

func (e *CommonEndpointer[H]) StartingAtVersion(version string) *CommonEndpointer[H] {
	ae := *e
	ae.startingAtVersion = version
	return &ae
}

func (e *CommonEndpointer[H]) EndingAtVersion(version string) *CommonEndpointer[H] {
	ae := *e
	ae.endingAtVersion = version
	return &ae
}

func (e *CommonEndpointer[H]) WithAltPaths(paths ...string) *CommonEndpointer[H] {
	ae := *e
	ae.alternativePaths = paths
	return &ae
}

func (e *CommonEndpointer[H]) WithCustomMiddleware(mws ...endpoint.Middleware) *CommonEndpointer[H] {
	ae := *e
	ae.CustomMiddleware = mws
	return &ae
}

func (e *CommonEndpointer[H]) UsePathPrefix() *CommonEndpointer[H] {
	ae := *e
	ae.usePathPrefix = true
	return &ae
}

// PathHandler registers a handler for the verb and path. The pathHandler is
// a function that receives the actual path to which it will be mounted, and
// returns the actual http.Handler that will handle this endpoint. This is for
// when the handler needs to know on which path it was called.
func (e *CommonEndpointer[H]) PathHandler(verb, path string, pathHandler func(path string) http.Handler) {
	e.HandlePathHandler(path, pathHandler, verb)
}

func (e *CommonEndpointer[H]) HandleHTTPHandler(path string, h http.Handler, verb string) {
	self := func(_ string) http.Handler { return h }
	e.HandlePathHandler(path, self, verb)
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

func (e *CommonEndpointer[H]) HandlePathHandler(path string, pathHandler func(path string) http.Handler, verb string) {
	versions := e.Versions
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
	if e.endingAtVersion == "" || e.endingAtVersion == e.Versions[len(e.Versions)-1] {
		versions = append(versions, "latest")
	}

	versionedPath := strings.Replace(path, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", strings.Join(versions, "|")), 1)
	nameAndVerb := getNameFromPathAndVerb(verb, path, e.startingAtVersion)
	if e.usePathPrefix {
		e.Router.PathPrefix(versionedPath).Handler(pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
	} else {
		e.Router.Handle(versionedPath, pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
	}
	for _, alias := range e.alternativePaths {
		nameAndVerb := getNameFromPathAndVerb(verb, alias, e.startingAtVersion)
		versionedPath := strings.Replace(alias, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", strings.Join(versions, "|")), 1)
		if e.usePathPrefix {
			e.Router.PathPrefix(versionedPath).Handler(pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
		} else {
			e.Router.Handle(versionedPath, pathHandler(versionedPath)).Name(nameAndVerb).Methods(verb)
		}
	}
}

func EncodeCommonResponse(
	ctx context.Context,
	w http.ResponseWriter,
	response interface{},
	jsonMarshal func(w http.ResponseWriter, response interface{}) error,
) error {
	if cs, ok := response.(cookieSetter); ok {
		cs.SetCookies(ctx, w)
	}

	// The has to happen first, if an error happens we'll redirect to an error
	// page and the error will be logged
	if page, ok := response.(htmlPage); ok {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		WriteBrowserSecurityHeaders(w)
		if coder, ok := page.Error().(kithttp.StatusCoder); ok {
			w.WriteHeader(coder.StatusCode())
		}
		_, err := io.WriteString(w, page.Html())
		return err
	}

	if e, ok := response.(fleet.Errorer); ok && e.Error() != nil {
		EncodeError(ctx, e.Error(), w)
		return nil
	}

	if render, ok := response.(renderHijacker); ok {
		render.HijackRender(ctx, w)
		return nil
	}

	if e, ok := response.(statuser); ok {
		w.WriteHeader(e.Status())
		if e.Status() == http.StatusNoContent {
			return nil
		}
	}

	return jsonMarshal(w, response)
}

// statuser allows response types to implement a custom
// http success status - default is 200 OK
type statuser interface {
	Status() int
}

// loads a html page
type htmlPage interface {
	Html() string
	Error() error
}

// renderHijacker can be implemented by response values to take control of
// their own rendering.
type renderHijacker interface {
	HijackRender(ctx context.Context, w http.ResponseWriter)
}

// cookieSetter can be implemented by response values to set cookies on the response.
type cookieSetter interface {
	SetCookies(ctx context.Context, w http.ResponseWriter)
}
