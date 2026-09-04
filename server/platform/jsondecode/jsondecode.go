// Package jsondecode decodes the JSON that Fleet validates strictly.
//
// It decodes exactly like encoding/json but reports errors through encoding/json/v2, because as of Go
// 1.27 that is the only way to learn *where* a decode failed. Under v1 error semantics an error returned
// from a custom UnmarshalJSON method is reported verbatim, so it reaches the caller with no position at
// all: UnmarshalTypeError.Field is empty.
package jsondecode

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// decodeOptions keeps decoding byte-for-byte compatible with encoding/json (case-insensitive member
// matching, duplicate names allowed, and so on) while turning off legacy error reporting, which is
// what makes the position of a failure available.
var decodeOptions = []jsonv2.Options{
	jsonv1.DefaultOptionsV1(),
	jsonv1.ReportErrorsWithLegacySemantics(false),
}

// RejectUnknownMembers is the equivalent of (*json.Decoder).DisallowUnknownFields.
func RejectUnknownMembers() jsonv2.Options { return jsonv2.RejectUnknownMembers(true) }

// Decoder reads JSON values from a stream, mirroring (*json.Decoder).
type Decoder struct {
	dec  *jsontext.Decoder
	opts []jsonv2.Options
}

// NewDecoder returns a Decoder reading from r.
func NewDecoder(r io.Reader, opts ...jsonv2.Options) *Decoder {
	// Built up rather than appended onto decodeOptions, which would risk writing into the shared slice.
	all := make([]jsonv2.Options, 0, len(decodeOptions)+len(opts))
	all = append(all, decodeOptions...)
	all = append(all, opts...)
	return &Decoder{dec: jsontext.NewDecoder(r, all...), opts: all}
}

// Decode reads the next JSON value from the stream into v. It returns io.EOF once the stream is
// exhausted, which callers use to detect trailing content.
func (d *Decoder) Decode(v any) error {
	return translate(jsonv2.UnmarshalDecode(d.dec, v, d.opts...), v)
}

// Unmarshal decodes the JSON value in data into v, rejecting trailing bytes like json.Unmarshal.
func Unmarshal(data []byte, v any) error {
	return translate(jsonv2.Unmarshal(data, v, decodeOptions...), v)
}

// translate rewrites a v2 error into the encoding/json error the v1 API would have produced, except that
// the field path is populated even when the failure came from a custom UnmarshalJSON. root is the value
// being decoded into; it supplies the type name UnmarshalTypeError.Struct reports, matching what Go
// 1.27's own v1 shim does.
//
// Anything else is returned unchanged, so io.EOF still ends a stream.
func translate(err error, root any) error {
	if _, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return syntaxError(err)
	}

	serr, ok := errors.AsType[*jsonv2.SemanticError](err)
	if err == nil || !ok {
		return err
	}

	// encoding/json reports an unrecognized member as `json: unknown field "x"`, and server/service,
	// server/fleet and pkg/spec all detect that case by matching the wording, so keep producing it.
	if errors.Is(serr.Err, jsonv2.ErrUnknownName) {
		return fmt.Errorf("json: unknown field %q", serr.JSONPointer.LastToken())
	}

	// A custom UnmarshalJSON that passes an encoding/json error straight back (every pkg/optjson type
	// does) has already named the concrete Go type that failed.
	//
	// The type assertion is deliberately direct rather than errors.As: a method that *wraps* an
	// UnmarshalTypeError has added a message of its own, and we want to keep that message.
	if inner, ok := serr.Err.(*jsonv1.UnmarshalTypeError); ok { //nolint:errorlint // see above
		inner.Struct = rootTypeName(root)
		inner.Field = FieldPath(err)
		return inner
	}

	// Anything else a custom UnmarshalJSON returned is a validation failure in its own right.
	if serr.Err != nil && !isNumberOverflow(serr) {
		return serr.Err
	}

	// A genuine type mismatch, which is the ordinary case and reaches here in one of two shapes. Usually
	// serr.Err is nil, because v2 spotted the mismatch itself -- a JSON string against a bool field, say
	// -- and nothing failed underneath it to report.
	return &jsonv1.UnmarshalTypeError{
		Value:  describeValue(serr),
		Type:   serr.GoType,
		Offset: serr.ByteOffset,
		Struct: rootTypeName(root),
		Field:  FieldPath(err),
	}
}

// errUnexpectedEOF carries encoding/json's wording for a truncated document while staying matchable as
// io.ErrUnexpectedEOF, which callers use to tell a truncated body apart from an exhausted size limit.
type errUnexpectedEOF struct{}

func (errUnexpectedEOF) Error() string { return "unexpected end of JSON input" }
func (errUnexpectedEOF) Unwrap() error { return io.ErrUnexpectedEOF }

// syntaxError restates a jsontext syntax error in encoding/json's wording.
func syntaxError(err error) error {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return errUnexpectedEOF{}
	}
	msg := err.Error()
	if i := strings.LastIndex(msg, "jsontext: "); i >= 0 {
		msg = msg[i+len("jsontext: "):]
	}
	return errors.New(msg)
}

// isNumberOverflow reports whether serr is v2 failing to fit a JSON number into a Go numeric type.
func isNumberOverflow(serr *jsonv2.SemanticError) bool {
	if !errors.Is(serr.Err, strconv.ErrSyntax) && !errors.Is(serr.Err, strconv.ErrRange) {
		return false
	}
	if serr.GoType == nil {
		return false
	}
	switch serr.GoType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// IsTypeError reports whether err describes a value of the wrong type, as opposed to malformed JSON or
// an unrelated failure.
func IsTypeError(err error) bool {
	_, ok := errors.AsType[*jsonv1.UnmarshalTypeError](err)
	return ok
}

// FieldPath returns the dot-separated path to the value that failed to decode, or "" when the position
// is unknown. Array elements appear as their index, so "specs.0.name" is the first element's name.
func FieldPath(err error) string {
	if serr, ok := errors.AsType[*jsonv2.SemanticError](err); ok && serr.JSONPointer != "" {
		var b strings.Builder
		for token := range serr.JSONPointer.Tokens() {
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(token)
		}
		return b.String()
	}

	if terr, ok := errors.AsType[*jsonv1.UnmarshalTypeError](err); ok {
		return terr.Field
	}
	return ""
}

// describeValue renders the offending JSON value the way UnmarshalTypeError.Value does: a bare kind
// ("bool", "string", "object"), except for numbers, which encoding/json spells out as "number 1.23".
func describeValue(serr *jsonv2.SemanticError) string {
	var value string
	switch serr.JSONKind {
	case 'n', '"', '0':
		value = serr.JSONKind.String()
	case 'f', 't':
		value = "bool"
	case '[', ']':
		value = "array"
	case '{', '}':
		value = "object"
	}
	if serr.JSONKind == '0' && len(serr.JSONValue) > 0 {
		value += " " + string(serr.JSONValue)
	}
	return value
}

func rootTypeName(v any) string {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil {
		return ""
	}
	return t.Name()
}
