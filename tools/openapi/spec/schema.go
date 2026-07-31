// Package spec assembles and validates the OpenAPI 3.1 document.
package spec

import (
	"math"
	"regexp"
)

var dateTimeRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)

// Infer derives a JSON Schema fragment from a decoded JSON example value.
// Examples can't express optionality or unions, so the schema is deliberately
// permissive: no required lists, and null-only values get the empty schema.
func Infer(v any) map[string]any {
	switch t := v.(type) {
	case nil:
		return map[string]any{}
	case bool:
		return map[string]any{"type": "boolean"}
	case string:
		if dateTimeRe.MatchString(t) {
			return map[string]any{"type": "string", "format": "date-time"}
		}
		return map[string]any{"type": "string"}
	case float64:
		if t == math.Trunc(t) {
			return map[string]any{"type": "integer"}
		}
		return map[string]any{"type": "number"}
	case []any:
		if len(t) == 0 {
			return map[string]any{"type": "array", "items": map[string]any{}}
		}
		return map[string]any{"type": "array", "items": Infer(t[0])}
	case map[string]any:
		props := make(map[string]any, len(t))
		for k, val := range t {
			props[k] = Infer(val)
		}
		return map[string]any{"type": "object", "properties": props}
	default:
		return map[string]any{}
	}
}
