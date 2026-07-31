package spec

import (
	"reflect"
	"testing"
)

func TestInferScalars(t *testing.T) {
	cases := []struct {
		in   any
		want map[string]any
	}{
		{"hello", map[string]any{"type": "string"}},
		{"2024-01-01T12:00:00Z", map[string]any{"type": "string", "format": "date-time"}},
		{float64(3), map[string]any{"type": "integer"}},
		{float64(3.5), map[string]any{"type": "number"}},
		{true, map[string]any{"type": "boolean"}},
		{nil, map[string]any{}},
	}
	for _, c := range cases {
		if got := Infer(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Infer(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestInferObjectAndArray(t *testing.T) {
	in := map[string]any{
		"id":   float64(1),
		"tags": []any{"a"},
		"nested": map[string]any{
			"ok": true,
		},
		"empty": []any{},
	}
	got := Infer(in)
	if got["type"] != "object" {
		t.Fatalf("want object, got %v", got)
	}
	props := got["properties"].(map[string]any)
	if props["id"].(map[string]any)["type"] != "integer" {
		t.Errorf("id: %v", props["id"])
	}
	tags := props["tags"].(map[string]any)
	if tags["type"] != "array" || tags["items"].(map[string]any)["type"] != "string" {
		t.Errorf("tags: %v", tags)
	}
	empty := props["empty"].(map[string]any)
	if empty["type"] != "array" || len(empty["items"].(map[string]any)) != 0 {
		t.Errorf("empty array should have permissive items: %v", empty)
	}
}

func TestInferNullField(t *testing.T) {
	in := map[string]any{"parent": nil}
	got := Infer(in)
	props := got["properties"].(map[string]any)
	if len(props["parent"].(map[string]any)) != 0 {
		t.Errorf("null-only field must be permissive {}: %v", props["parent"])
	}
}
