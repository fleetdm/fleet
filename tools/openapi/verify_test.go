package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
	if !strings.Contains(result.Detail, "  at ") {
		t.Errorf("want detailed schema failure lines (\"  at <location>: <reason>\"), got %q", result.Detail)
	}
}

func TestCheckMDMEndpointTreatsMDMOffAsPartial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "MDM features aren't turned on in Fleet. For more information about setting up MDM, please visit https://fleetdm.com/docs/using-fleet/mobile-device-management"}`))
	}))
	defer srv.Close()

	v, err := newSpecValidator(testSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	result := checkMDMEndpoint(v, srv.URL, "tok", "GET", "/api/v1/fleet/widgets", nil)
	if result.Status != statusPartial {
		t.Fatalf("want partial, got %+v", result)
	}
}

func TestCheckMDMEndpointFailsOnOtherErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "some other bad request"}`))
	}))
	defer srv.Close()

	v, err := newSpecValidator(testSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	result := checkMDMEndpoint(v, srv.URL, "tok", "GET", "/api/v1/fleet/widgets", nil)
	if result.Status != statusFailed {
		t.Fatalf("want failed, got %+v", result)
	}
}

func TestCheckMDMEndpointValidatesHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"widgets": [{"id": 2, "name": "hammer"}]}`))
	}))
	defer srv.Close()

	v, err := newSpecValidator(testSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	result := checkMDMEndpoint(v, srv.URL, "tok", "GET", "/api/v1/fleet/widgets", nil)
	if result.Status != statusVerified {
		t.Fatalf("want verified, got %+v", result)
	}
}
