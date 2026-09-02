// Package jsondecode recovers the field path from JSON decode errors.
//
// As of Go 1.27 encoding/json is implemented on top of encoding/json/v2. Under v1 error semantics an
// error returned from a custom UnmarshalJSON method is reported verbatim, so it reaches the caller with
// no position information: UnmarshalTypeError.Field is empty. Every pkg/optjson type has such a method,
// which is why "deadline_days must be a number" degraded into an error that could not say which field
// it meant.
//
// The position is only available under v2 error semantics, and that setting is not separable from v2
// decoding behavior -- the same option also makes a decode abort at the first error instead of
// continuing, which callers such as the GitOps "force" path depend on. So rather than change how Fleet
// decodes, Enrich decodes a second time purely to locate the failure, and only when the first decode
// already failed without a position. Decoding behavior is therefore completely unchanged.
package jsondecode

import (
	jsonv1 "encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"reflect"
	"strings"
)

// v2Errors decodes exactly like encoding/json but reports errors with v2's structure, which is the only
// form that carries the position of the failure.
var v2Errors = []jsonv2.Options{
	jsonv1.DefaultOptionsV1(),
	jsonv1.ReportErrorsWithLegacySemantics(false),
}

// Enrich annotates a decode error with the path of the value that failed, for the cases where Go can no
// longer report it: err must be an *encoding/json.UnmarshalTypeError with an empty Field. data is the
// input that produced err and v is the value it was decoded into; both are needed to locate the failure
// again. Any other error, and any error that already names a field, is returned unchanged.
//
// The returned error is the same *UnmarshalTypeError, annotated in place, which is what the pre-Go 1.27
// decoder did. Callers that resolve an error to its root cause therefore keep working.
func Enrich(err error, data []byte, v any) error {
	terr, ok := errors.AsType[*jsonv1.UnmarshalTypeError](err)
	if err == nil || !ok || terr.Field != "" {
		return err
	}

	// Decode again into a throwaway value of the same type. Both decoders report the first error in the
	// input, so this locates the same failure the caller already saw.
	scratch := reflect.New(indirectType(reflect.TypeOf(v)))
	path := FieldPath(jsonv2.Unmarshal(data, scratch.Interface(), v2Errors...))
	if path == "" {
		return err
	}

	terr.Struct = rootTypeName(v)
	terr.Field = path
	return err
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

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func rootTypeName(v any) string {
	t := indirectType(reflect.TypeOf(v))
	if t == nil {
		return ""
	}
	return t.Name()
}
