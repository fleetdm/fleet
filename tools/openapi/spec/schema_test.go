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
		{"hello", map[string]any{"type": []any{"string", "null"}}},
		{"2024-01-01T12:00:00Z", map[string]any{"type": []any{"string", "null"}, "format": "date-time"}},
		{float64(3), map[string]any{"type": []any{"integer", "null"}}},
		{float64(3.5), map[string]any{"type": []any{"number", "null"}}},
		{true, map[string]any{"type": []any{"boolean", "null"}}},
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
	if !reflect.DeepEqual(got["type"], []any{"object", "null"}) {
		t.Fatalf("want nullable object, got %v", got)
	}
	props := got["properties"].(map[string]any)
	if !reflect.DeepEqual(props["id"].(map[string]any)["type"], []any{"integer", "null"}) {
		t.Errorf("id: %v", props["id"])
	}
	tags := props["tags"].(map[string]any)
	if !reflect.DeepEqual(tags["type"], []any{"array", "null"}) ||
		!reflect.DeepEqual(tags["items"].(map[string]any)["type"], []any{"string", "null"}) {
		t.Errorf("tags: %v", tags)
	}
	empty := props["empty"].(map[string]any)
	if !reflect.DeepEqual(empty["type"], []any{"array", "null"}) || len(empty["items"].(map[string]any)) != 0 {
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
