package endpointer

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/platform/middleware/authzcheck"
	"github.com/fleetdm/fleet/v4/server/platform/middleware/ratelimit"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// We use our own wrapper here, to preserve the Close method of the original io.ReadCloser
// But also allows us to modify the limit at a laterp oint.
type LimitedReadCloser struct {
	*io.LimitedReader
	Closer io.Closer
}

func (lrc *LimitedReadCloser) Close() error {
	return lrc.Closer.Close()
}

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

// SwapRules returns a new slice with OldKey and NewKey swapped in each rule.
// This is useful for output renaming: the struct marshals with OldKey names,
// and we want to rewrite them to NewKey names.
func SwapRules(rules []AliasRule) []AliasRule {
	swapped := make([]AliasRule, len(rules))
	for i, r := range rules {
		swapped[i] = AliasRule{OldKey: r.NewKey, NewKey: r.OldKey}
	}
	return swapped
}

// aliasRulesCache caches the result of ExtractAliasRules by reflect.Type so
// that the reflection walk happens only once per struct type, not on every
// request.
var aliasRulesCache sync.Map // reflect.Type → []AliasRule

// ExtractAliasRules inspects the struct type of iface (recursively, including
// embedded structs) and builds an []AliasRule from fields that carry a
// `renameto` struct tag. For each such field the json tag's field name
// becomes OldKey (the current/deprecated name) and the renameto value becomes
// NewKey (the target name).
//
// Only `json` tags are considered; `url` and `query` tags are ignored for now.
//
// The returned slice is deduplicated: if the same alias pair appears on
// multiple fields (e.g. in both a request and an embedded struct) it is
// included only once.
//
// Results are cached by type so that the reflection walk only happens once.
func ExtractAliasRules(iface any) []AliasRule {
	if iface == nil {
		return nil
	}
	t := reflect.TypeOf(iface)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	if cached, ok := aliasRulesCache.Load(t); ok {
		return cached.([]AliasRule)
	}

	seen := make(map[AliasRule]bool)
	var rules []AliasRule
	extractAliasRulesFromType(t, seen, &rules)
	aliasRulesCache.Store(t, rules)
	return rules
}

func extractAliasRulesFromType(t reflect.Type, seen map[AliasRule]bool, rules *[]AliasRule) {
	// visited tracks types we've already walked to avoid infinite recursion
	// from cyclic type references (e.g. type Node struct { Children []Node }).
	visited := make(map[reflect.Type]bool)
	extractAliasRulesRecursive(t, seen, rules, visited)
}

// elemType dereferences pointer, slice, array, and map types to find the
// underlying (possibly struct) element type.
func elemType(t reflect.Type) reflect.Type {
	for {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			t = t.Elem()
		case reflect.Map:
			// For maps, the values may contain structs with renameto tags.
			t = t.Elem()
		default:
			return t
		}
	}
}

func extractAliasRulesRecursive(t reflect.Type, seen map[AliasRule]bool, rules *[]AliasRule, visited map[reflect.Type]bool) {
	if visited[t] {
		return
	}
	visited[t] = true

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		// Check this field for a renameto tag.
		renameTo, hasRenameTo := sf.Tag.Lookup("renameto")
		if hasRenameTo && renameTo != "" {
			jsonTag, hasJSON := sf.Tag.Lookup("json")
			if hasJSON && jsonTag != "" && jsonTag != "-" {
				// Strip options like ",omitempty" from the json tag.
				jsonFieldName, _, _ := strings.Cut(jsonTag, ",")
				if jsonFieldName != "" && jsonFieldName != "-" {
					rule := AliasRule{OldKey: jsonFieldName, NewKey: renameTo}
					if !seen[rule] {
						seen[rule] = true
						*rules = append(*rules, rule)
					}
				}
			}
		}

		// Recurse into any struct type reachable from this field
		// (through pointers, slices, arrays, maps, or directly).
		ft := elemType(sf.Type)
		if ft.Kind() == reflect.Struct {
			extractAliasRulesRecursive(ft, seen, rules, visited)
		}
	}
}

func BadRequestErr(publicMsg string, internalErr error) error {
	// ensure timeout errors don't become BadRequestErrors.
	var opErr *net.OpError
	if errors.As(internalErr, &opErr) {
		return fmt.Errorf(publicMsg+", internal: %w", internalErr)
	}
	return &platform_http.BadRequestError{
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

// DomainQueryFieldDecoder decodes a query parameter value into the target field.
// It returns true if it handled the field, false if default handling should be used.
type DomainQueryFieldDecoder func(queryTagName, queryVal string, field reflect.Value) (handled bool, err error)

func DecodeQueryTagValue(r *http.Request, fp fieldPair, customDecoder DomainQueryFieldDecoder, ctx context.Context) error {
	queryTagValue, ok := fp.Sf.Tag.Lookup("query")

	if ok {
		var err error
		var optional bool
		queryTagValue, optional, err = ParseTag(queryTagValue)
		if err != nil {
			return err
		}
		queryVal := r.URL.Query().Get(queryTagValue)

		// The query tag now holds the old (deprecated) name. If the old name
		// was used, log a deprecation warning. If not found, check the
		// renameto value (the new name) as a fallback.
		if queryVal != "" {
			if renameTo, hasRenameTo := fp.Sf.Tag.Lookup("renameto"); hasRenameTo {
				// Check for conflict: if both old and new names are provided, return an error.
				newName, _, _ := ParseTag(renameTo)
				if newVal := r.URL.Query().Get(newName); newVal != "" {
					return &platform_http.BadRequestError{
						Message: fmt.Sprintf("Specify only one of %q or %q", queryTagValue, newName),
					}
				}
				// Log deprecation warning - the old name was used.
				logging.WithLevel(ctx, slog.LevelWarn)
				logging.WithExtras(ctx,
					"deprecated_param", queryTagValue,
					"deprecation_warning", fmt.Sprintf("'%s' is deprecated, use '%s' instead", queryTagValue, renameTo),
				)
			}
		} else {
			if renameTo, hasRenameTo := fp.Sf.Tag.Lookup("renameto"); hasRenameTo {
				renameTo, _, err = ParseTag(renameTo)
				if err != nil {
					return err
				}
				queryVal = r.URL.Query().Get(renameTo)
			}
		}
		// If we still don't have a value, return if this is optional, otherwise error.
		if queryVal == "" {
			if optional {
				return nil
			}
			return &platform_http.BadRequestError{Message: fmt.Sprintf("Param %s is required", queryTagValue)}
		}
		field := fp.V
		if field.Kind() == reflect.Ptr {
			// create the new instance of whatever it is
			field.Set(reflect.New(field.Type().Elem()))
			field = field.Elem()
		}

		// Try custom decoder first if provided
		if customDecoder != nil {
			handled, err := customDecoder(queryTagValue, queryVal, field)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
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
			queryValInt, err := strconv.Atoi(queryVal)
			if err != nil {
				return BadRequestErr("parsing int from query", err)
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

func extractIP(r *http.Request) string {
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
	Logger *slog.Logger
}

func (h *ErrorHandler) Handle(ctx context.Context, err error) {
	path, _ := ctx.Value(kithttp.ContextKeyRequestPath).(string)

	attrs := []any{"path", path}

	if startTime, ok := logging.StartTime(ctx); ok && !startTime.IsZero() {
		attrs = append(attrs, "took", time.Since(startTime))
	}

	var ewi platform_http.ErrWithInternal
	if errors.As(err, &ewi) {
		attrs = append(attrs, "internal", ewi.Internal())
	}

	var ewlf platform_http.ErrWithLogFields
	if errors.As(err, &ewlf) {
		attrs = append(attrs, ewlf.LogFields()...)
	}

	var uuider platform_http.ErrorUUIDer
	if errors.As(err, &uuider) {
		attrs = append(attrs, "uuid", uuider.UUID())
	}

	var rle ratelimit.Error
	if errors.As(err, &rle) {
		res := rle.Result()
		if res.RetryAfter > 0 {
			attrs = append(attrs, "retry_after", res.RetryAfter)
		}
		attrs = append(attrs, "err", "limit exceeded")
	} else {
		attrs = append(attrs, "err", err)
	}

	h.Logger.InfoContext(ctx, "request error", attrs...)
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
// struct has at least 1 json tag it'll unmarshall the body. Custom `url` tag
// values can be handled by providing a parseCustomTags function. Note that
// these behaviors do not work for embedded structs.
//
// Any other `url` tag will be treated as a path variable (of the form
// /path/{name} in the route's path) from the URL path pattern, and it'll be
// decoded and set accordingly. Variables can be optional by setting the tag as
// follows: `url:"some-id,optional"`.
//
// If iface implements the RequestDecoder interface, it returns a function that
// calls iface.DecodeRequest(ctx, r) - i.e. the value itself fully controls its
// own decoding.
//
// If iface implements the bodyDecoder interface, it calls iface.DecodeBody
// after having decoded any non-body fields (such as url and query parameters)
// into the struct.
//
// The customQueryDecoder parameter allows services to inject domain-specific
// query parameter decoding logic.
//
// If adding a new way to parse/decode the requset, make sure to wrap the body in a limited reader with the maxRequestBodySize
func MakeDecoder(
	iface interface{},
	jsonUnmarshal func(body io.Reader, req any) error,
	parseCustomTags func(urlTagValue string, r *http.Request, field reflect.Value) (bool, error),
	isBodyDecoder func(reflect.Value) bool,
	decodeBody func(ctx context.Context, r *http.Request, v reflect.Value, body io.Reader) error,
	customQueryDecoder DomainQueryFieldDecoder,
	maxRequestBodySize int64,
) kithttp.DecodeRequestFunc {
	// Infer alias rules from `renameto` struct tags on the request type.
	aliasRules := ExtractAliasRules(iface)
	if iface == nil {
		return func(ctx context.Context, r *http.Request) (interface{}, error) {
			return nil, nil
		}
	}
	if rd, ok := iface.(RequestDecoder); ok {
		return func(ctx context.Context, r *http.Request) (interface{}, error) {
			if maxRequestBodySize != -1 {
				limitedReader := io.LimitReader(r.Body, maxRequestBodySize).(*io.LimitedReader)

				r.Body = &LimitedReadCloser{
					LimitedReader: limitedReader,
					Closer:        r.Body,
				}
			}
			ret, err := rd.DecodeRequest(ctx, r)
			if err != nil && errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, platform_http.PayloadTooLargeError{ContentLength: r.Header.Get("Content-Length"), MaxRequestSize: maxRequestBodySize}
			}
			return ret, err
		}
	}

	t := reflect.TypeOf(iface)
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("MakeDecoder only understands structs, not %T", iface))
	}

	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		v := reflect.New(t)
		nilBody := false
		var rewriter *JSONKeyRewriteReader

		if maxRequestBodySize != -1 {
			limitedReader := io.LimitReader(r.Body, maxRequestBodySize).(*io.LimitedReader)

			r.Body = &LimitedReadCloser{
				LimitedReader: limitedReader,
				Closer:        r.Body,
			}
		}

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

			// Insert the JSON key rewriter into the reader pipeline
			// (after gzip decompression, before JSON decoding) to rename
			// deprecated field names and detect alias conflicts.
			if len(aliasRules) > 0 {
				rewriter = NewJSONKeyRewriteReader(body, aliasRules)
				//nolint:errcheck // nothing to do on .Close() error.
				defer rewriter.Close()
				body = rewriter
			}

			if isBodyDecoder == nil || !isBodyDecoder(v) {
				req := v.Interface()
				err := jsonUnmarshal(body, req)
				if err != nil {
					// Check for alias conflict errors from the rewriter.
					var ace *AliasConflictError
					if errors.As(err, &ace) {
						return nil, &platform_http.BadRequestError{
							Message:     fmt.Sprintf("Specify only one of %q or %q", ace.Old, ace.New),
							InternalErr: ace,
						}
					}

					if errors.Is(err, io.ErrUnexpectedEOF) {
						return nil, platform_http.PayloadTooLargeError{ContentLength: r.Header.Get("Content-Length"), MaxRequestSize: maxRequestBodySize}
					}

					return nil, BadRequestErr("json decoder error", err)
				}
				v = reflect.ValueOf(req)
			}

			// Log deprecation warnings when deprecated field names are used.
			if rewriter != nil {
				if deprecated := rewriter.UsedDeprecatedKeys(); len(deprecated) > 0 {
					newNames := make([]string, len(deprecated))
					for i, old := range deprecated {
						for _, rule := range aliasRules {
							if rule.OldKey == old {
								newNames[i] = rule.NewKey
								break
							}
						}
					}
					logging.WithLevel(ctx, slog.LevelWarn)
					logging.WithExtras(ctx,
						"deprecated_fields", fmt.Sprintf("%v", deprecated),
						"deprecation_warning", fmt.Sprintf("use the updated field names (%s) instead", newNames),
					)
				}
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
				return nil, &platform_http.BadRequestError{Message: "Expected JSON Body"}
			}

			isContentJson := r.Header.Get("Content-Type") == "application/json"
			isCrossSite := r.Header.Get("Origin") != "" || r.Header.Get("Referer") != ""
			if jsonExpected && isCrossSite && !isContentJson {
				return nil, platform_http.NewUserMessageError(errors.New("Expected Content-Type \"application/json\""), http.StatusUnsupportedMediaType)
			}

			err = DecodeQueryTagValue(r, fp, customQueryDecoder, ctx)
			if err != nil {
				return nil, err
			}
		}

		if isBodyDecoder != nil && isBodyDecoder(v) {
			err := decodeBody(ctx, r, v, body)
			if err != nil {
				// Check for alias conflict errors from the rewriter.
				var ace *AliasConflictError
				if errors.As(err, &ace) {
					return nil, &platform_http.BadRequestError{
						Message:     fmt.Sprintf("Specify only one of %q or %q", ace.Old, ace.New),
						InternalErr: ace,
					}
				}

				if errors.Is(err, io.ErrUnexpectedEOF) {
					return nil, platform_http.PayloadTooLargeError{ContentLength: r.Header.Get("Content-Length"), MaxRequestSize: maxRequestBodySize}
				}
				return nil, err
			}

			// Log deprecation warnings when deprecated field names are used
			// (bodyDecoder path).
			if rewriter != nil {
				if deprecated := rewriter.UsedDeprecatedKeys(); len(deprecated) > 0 {
					logging.WithLevel(ctx, slog.LevelWarn)
					logging.WithExtras(ctx,
						"deprecated_fields", fmt.Sprintf("%v", deprecated),
						"deprecation_warning", "use the updated field names instead",
					)
				}
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
						return nil, &platform_http.BadRequestError{Message: fmt.Sprintf(
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

type CommonEndpointer[H any] struct {
	EP            Endpointer[H]
	MakeDecoderFn func(iface any, requestBodyLimit int64) kithttp.DecodeRequestFunc
	EncodeFn      kithttp.EncodeResponseFunc
	Opts          []kithttp.ServerOption
	Router        *mux.Router
	Versions      []string

	// AuthMiddleware is a pre-built authentication middleware.
	AuthMiddleware endpoint.Middleware

	// CustomMiddleware are middlewares that run before authentication.
	CustomMiddleware []endpoint.Middleware
	// CustomMiddlewareAfterAuth are middlewares that run after authentication.
	CustomMiddlewareAfterAuth []endpoint.Middleware

	startingAtVersion string
	endingAtVersion   string
	alternativePaths  []string
	usePathPrefix     bool

	// The limit of the request body size in bytes, if set to -1 there is no limit.
	requestBodySizeLimit int64
}

type Endpointer[H any] interface {
	CallHandlerFunc(f H, ctx context.Context, request any, svc any) (platform_http.Errorer, error)
	Service() any
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

	// Apply "after auth" middleware (in reverse order so that the first wraps
	// the second wraps the third etc.)
	endp := next
	if len(e.CustomMiddlewareAfterAuth) > 0 {
		for i := len(e.CustomMiddlewareAfterAuth) - 1; i >= 0; i-- {
			mw := e.CustomMiddlewareAfterAuth[i]
			endp = mw(endp)
		}
	}
	if e.AuthMiddleware == nil {
		// This panic catches potential security issues during development.
		panic("AuthMiddleware must be set on CommonEndpointer")
	}
	endp = e.AuthMiddleware(endp)

	// Apply "before auth" middleware (in reverse order so that the first wraps
	// the second wraps the third etc.)
	for i := len(e.CustomMiddleware) - 1; i >= 0; i-- {
		mw := e.CustomMiddleware[i]
		endp = mw(endp)
	}

	// Default to MaxRequestBodySize if no limit is set, this ensures no endpointers are forgot
	// -1 = no limit, so don't default to anything if that is set, which can only be set with the appropriate SKIP method.
	if e.requestBodySizeLimit != -1 && (e.requestBodySizeLimit == 0 || e.requestBodySizeLimit < platform_http.MaxRequestBodySize) {
		// If no value is configured set default, or if the set endpoint value is less than global default use default.
		e.requestBodySizeLimit = platform_http.MaxRequestBodySize
	}
	return newServer(endp, e.MakeDecoderFn(v, e.requestBodySizeLimit), e.EncodeFn, e.Opts)
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

func (e *CommonEndpointer[H]) AppendCustomMiddleware(mws ...endpoint.Middleware) *CommonEndpointer[H] {
	ae := *e
	ae.CustomMiddleware = append(ae.CustomMiddleware, mws...)
	return &ae
}

func (e *CommonEndpointer[H]) WithCustomMiddlewareAfterAuth(mws ...endpoint.Middleware) *CommonEndpointer[H] {
	ae := *e
	ae.CustomMiddlewareAfterAuth = mws
	return &ae
}

func (e *CommonEndpointer[H]) UsePathPrefix() *CommonEndpointer[H] {
	ae := *e
	ae.usePathPrefix = true
	return &ae
}

func (e *CommonEndpointer[H]) WithRequestBodySizeLimit(limit int64) *CommonEndpointer[H] {
	ae := *e
	if limit > 0 {
		// Only set it when the limit is more than 0
		ae.requestBodySizeLimit = limit
	}
	return &ae
}

func (e *CommonEndpointer[H]) SkipRequestBodySizeLimit() *CommonEndpointer[H] {
	ae := *e
	ae.requestBodySizeLimit = -1
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
	domainErrorEncoder DomainErrorEncoder,
) error {
	// Infer alias rules from `renameto` struct tags on the response type.
	aliasRules := ExtractAliasRules(response)
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

	if e, ok := response.(platform_http.Errorer); ok && e.Error() != nil {
		EncodeError(ctx, e.Error(), w, domainErrorEncoder)
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

	// If alias rules are configured, buffer the JSON output so we can
	// duplicate keys (old→new) for forwards compatibility before writing
	// to the response.
	if len(aliasRules) > 0 {
		var buf bytes.Buffer
		bufWriter := &bufferedResponseWriter{ResponseWriter: w, buf: &buf}
		if err := jsonMarshal(bufWriter, response); err != nil {
			return err
		}
		transformed := DuplicateJSONKeys(buf.Bytes(), aliasRules)
		_, err := w.Write(transformed)
		return err
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

// bufferedResponseWriter wraps an http.ResponseWriter but redirects Write
// calls to a bytes.Buffer, allowing the output to be captured and
// transformed before being sent to the real writer. It implements
// http.ResponseWriter so it can be passed to jsonMarshal functions.
type bufferedResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (b *bufferedResponseWriter) Write(data []byte) (int, error) {
	return b.buf.Write(data)
}
