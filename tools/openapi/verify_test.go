package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/tools/openapi/parser"
	"github.com/fleetdm/fleet/tools/openapi/spec"
)

func testSpec(t *testing.T) []byte {
	t.Helper()
	res := &parser.Result{Endpoints: []parser.Endpoint{{
		Section: "Widgets", Heading: "List widgets", Method: "GET",
		Path: "/api/v1/fleet/widgets",
		Responses: []parser.Response{{Status: 200, Example: map[string]any{
			"widgets": []any{map[string]any{"id": float64(1), "name": "wrench"}},
		}}},
	}}}
	allow := spec.Allowlist{Endpoints: []spec.AllowedEndpoint{{Method: "GET", Path: "/api/v1/fleet/widgets"}}}
	doc, err := spec.Build(res, allow)
	if err != nil {
		t.Fatal(err)
	}
	b, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCheckEndpointAgainstSpec(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"widgets": [{"id": 2, "name": "hammer"}]}`))
	}))
	defer srv.Close()

	v, err := newSpecValidator(testSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	result := checkEndpoint(v, srv.URL, "tok", "GET", "/api/v1/fleet/widgets", nil)
	if result.Status != statusVerified {
		t.Fatalf("want verified, got %+v", result)
	}
}

func TestCheckEndpointDetectsShapeMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"widgets": "not-an-array"}`))
	}))
	defer srv.Close()

	v, err := newSpecValidator(testSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	result := checkEndpoint(v, srv.URL, "tok", "GET", "/api/v1/fleet/widgets", nil)
	if result.Status != statusFailed {
		t.Fatalf("want failed, got %+v", result)
	}
}
