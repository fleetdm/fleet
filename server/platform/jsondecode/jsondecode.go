// Package jsondecode is how Fleet decodes JSON request bodies and spec files.
//
// It decodes exactly like encoding/json but reports errors through encoding/json/v2, because as of Go
// 1.27 that is the only way to learn *where* a decode failed. Under v1 error semantics an error returned
// from a custom UnmarshalJSON method is reported verbatim, so it reaches the caller with no position at
// all: UnmarshalTypeError.Field is empty. Every pkg/optjson type has such a method, which is why
// "deadline_days must be a number" degraded into an error that could not say which field it meant.
//
// Callers never see a v2 error. Everything is translated back into the encoding/json error values Fleet
// already matches on, with the field path filled in, so consumers such as platform/http.UserMessageError,
// pkg/spec.ParseTypeError and the "unknown field" checks in server/service and server/fleet keep working
// unchanged.
//
// The one behavior difference is that v2 stops decoding at the first error where v1 kept going and
// reported the first error at the end. That only matters to a caller that reads a partially decoded
// value after a failure. server/service.applyTeamSpecsRequest.DecodeBody is the only such caller, and it
// re-decodes leniently when it wants to accept a spec despite an unknown field.
package jsondecode

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// decodeOptions keeps decoding byte-for-byte compatible with encoding/json -- case-insensitive member
// matching, duplicate names allowed, and so on -- while turning off legacy error reporting, which is
// what makes the position of a failure available.
var decodeOptions = []jsonv2.Options{
	jsonv1.DefaultOptionsV1(),
	jsonv1.ReportErrorsWithLegacySemantics(false),
}

// RejectUnknownMembers is the equivalent of (*json.Decoder).DisallowUnknownFields.
func RejectUnknownMembers() jsonv2.Options { return jsonv2.RejectUnknownMembers(true) }

// Decoder reads JSON values from a stream, mirroring (*json.Decoder). Callers reading more than one
// value from the same reader need this rather than repeated Decode calls, which would each start over.
type Decoder struct {
	dec  *jsontext.Decoder
	opts []jsonv2.Options
}

// NewDecoder returns a Decoder reading from r.
func NewDecoder(r io.Reader, opts ...jsonv2.Options) *Decoder {
	all := combine(opts)
	return &Decoder{dec: jsontext.NewDecoder(r, all...), opts: all}
}

// Decode reads the next JSON value from the stream into v. It returns io.EOF once the stream is
// exhausted, which callers use to detect trailing content.
func (d *Decoder) Decode(v any) error {
	return translate(jsonv2.UnmarshalDecode(d.dec, v, d.opts...), v)
}

// Decode reads a single JSON value from r into v. Like (*json.Decoder).Decode, and unlike
// json.Unmarshal, bytes after the value are left in r rather than rejected.
func Decode(r io.Reader, v any, opts ...jsonv2.Options) error {
	return NewDecoder(r, opts...).Decode(v)
}

// Unmarshal decodes the JSON value in data into v, rejecting trailing bytes like json.Unmarshal.
func Unmarshal(data []byte, v any, opts ...jsonv2.Options) error {
	all := combine(opts)
	return translate(jsonv2.Unmarshal(data, v, all...), v)
}

func combine(extra []jsonv2.Options) []jsonv2.Options {
	all := make([]jsonv2.Options, 0, len(decodeOptions)+len(extra))
	all = append(all, decodeOptions...)
	return append(all, extra...)
}

// translate rewrites a v2 error into the encoding/json error the v1 API would have produced, except that
// the field path is populated even when the failure came from a custom UnmarshalJSON. root is the value
// being decoded into; it supplies the type name UnmarshalTypeError.Struct reports, matching what Go
// 1.27's own v1 shim does. Anything that is not a v2 semantic error is returned unchanged, so io.EOF and
// syntax errors reach callers as they always did.
func translate(err error, root any) error {
	serr, ok := errors.AsType[*jsonv2.SemanticError](err)
	if err == nil || !ok {
		return err
	}

	// encoding/json reports an unrecognized member as `json: unknown field "x"`, and server/service,
	// server/fleet and pkg/spec all detect that case by matching the wording, so keep producing it.
	if errors.Is(serr.Err, jsonv2.ErrUnknownName) {
		return fmt.Errorf("json: unknown field %q", serr.JSONPointer.LastToken())
	}

	// A custom UnmarshalJSON that delegates to encoding/json (every pkg/optjson type) has already
	// produced an UnmarshalTypeError naming the concrete Go type that failed -- int, say, rather than the
	// optjson.Int wrapping it. Annotate that error with the position rather than wrapping it, which is
	// what the pre-Go 1.27 decoder did and keeps callers that walk to the root cause working.
	if inner, ok := errors.AsType[*jsonv1.UnmarshalTypeError](serr.Err); ok {
		inner.Struct = rootTypeName(root)
		inner.Field = FieldPath(err)
		return inner
	}

	// Err is deliberately left unset: callers resolve errors to their root cause, so wrapping the v2
	// error here would hide this one and defeat the point of reporting a position at all.
	return &jsonv1.UnmarshalTypeError{
		Value:  describeValue(serr),
		Type:   serr.GoType,
		Offset: serr.ByteOffset,
		Struct: rootTypeName(root),
		Field:  FieldPath(err),
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
