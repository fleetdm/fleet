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

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit"
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

type FieldPair struct {
	Sf reflect.StructField
	V  reflect.Value
}

// AllFields returns all the fields for a struct, including the ones from embedded structs
func AllFields(ifv reflect.Value) []FieldPair {
	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
	}
	if ifv.Kind() != reflect.Struct {
		return nil
	}

	var fields []FieldPair

	if !ifv.IsValid() {
		return nil
	}

	t := ifv.Type()

	for i := 0; i < ifv.NumField(); i++ {
		v := ifv.Field(i)

		if v.Kind() == reflect.Struct && t.Field(i).Anonymous {
			fields = append(fields, AllFields(v)...)
			continue
		}
		fields = append(fields, FieldPair{Sf: ifv.Type().Field(i), V: v})
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

func DecodeQueryTagValue(r *http.Request, fp FieldPair) error {
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
var trueClientIP = http.CanonicalHeaderKey("True-Client-IP")
var xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIP = http.CanonicalHeaderKey("X-Real-IP")

func ExtractIP(r *http.Request) string {
	ip := r.RemoteAddr
	if i := strings.LastIndexByte(ip, ':'); i != -1 {
		ip = ip[:i]
	}

	if tcip := r.Header.Get(trueClientIP); tcip != "" {
		ip = tcip
	} else if xrip := r.Header.Get(xRealIP); xrip != "" {
		ip = xrip
	} else if xff := r.Header.Get(xForwardedFor); xff != "" {
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

// A value that implements requestDecoder takes control of decoding the request
// as a whole - that is, it is responsible for decoding the body and any url
// or query argument itself.
type requestDecoder interface {
	DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error)
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
// If iface implements the requestDecoder interface, it returns a function that
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
	if rd, ok := iface.(requestDecoder); ok {
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

		fields := AllFields(v)
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

		return v.Interface(), nil
	}
}
