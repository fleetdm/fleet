package spec

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/tools/openapi/parser"
)

func sampleResult() *parser.Result {
	return &parser.Result{
		Endpoints: []parser.Endpoint{
			{
				Section: "Widgets", Heading: "List widgets", Method: "GET",
				Path:        "/api/v1/fleet/widgets",
				Description: "Returns all widgets.",
				Params: []parser.Param{
					{Name: "page", Type: "integer", In: "query", Description: "Page number."},
				},
				Responses: []parser.Response{
					{Status: 200, Example: map[string]any{"widgets": []any{}}},
				},
			},
			{
				Section: "Widgets", Heading: "Make widget", Method: "POST",
				Path: "/api/v1/fleet/widgets",
				Params: []parser.Param{
					{Name: "name", Type: "string", In: "body", Description: "The name.", Required: true},
					{Name: "count", Type: "integer", In: "body", Description: "How many."},
				},
				Responses: []parser.Response{
					{Status: 200, Example: map[string]any{"widget": map[string]any{"id": float64(2)}}},
				},
			},
		},
	}
}

func allowAll() Allowlist {
	return Allowlist{Endpoints: []AllowedEndpoint{
		{Method: "GET", Path: "/api/v1/fleet/widgets"},
		{Method: "POST", Path: "/api/v1/fleet/widgets"},
	}}
}

func TestBuildEmitsAllowlistedOperations(t *testing.T) {
	doc, err := Build(sampleResult(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	item, ok := doc.Paths["/api/v1/fleet/widgets"]
	if !ok {
		t.Fatalf("path missing: %v", doc.Paths)
	}
	get, ok := item["get"]
	if !ok || get.OperationID != "listWidgets" {
		t.Fatalf("get op wrong: %+v", item)
	}
	if len(get.Parameters) != 1 || get.Parameters[0].Name != "page" {
		t.Errorf("parameters wrong: %+v", get.Parameters)
	}
	post := item["post"]
	if post.RequestBody == nil {
		t.Fatal("post has no request body")
	}
	schema := post.RequestBody.Content["application/json"].Schema
	req := schema["required"].([]string)
	if len(req) != 1 || req[0] != "name" {
		t.Errorf("required wrong: %v", req)
	}
}

func TestBuildFailsOnMissingAllowlistedEndpoint(t *testing.T) {
	allow := Allowlist{Endpoints: []AllowedEndpoint{{Method: "GET", Path: "/api/v1/fleet/nope"}}}
	_, err := Build(sampleResult(), allow)
	if err == nil || !strings.Contains(err.Error(), "/api/v1/fleet/nope") {
		t.Fatalf("want error naming missing path, got %v", err)
	}
}

func TestBuildFailsOnSkippedAllowlistedEndpoint(t *testing.T) {
	res := sampleResult()
	res.Endpoints = res.Endpoints[1:]
	res.Skipped = append(res.Skipped, parser.Skip{Heading: "List widgets", Reason: "no request line found"})
	_, err := Build(res, allowAll())
	if err == nil {
		t.Fatal("want error for skipped allowlisted endpoint")
	}
}

func TestRenderDeterministic(t *testing.T) {
	doc, err := Build(sampleResult(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	a, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := doc.Render()
	if string(a) != string(b) {
		t.Error("marshal not deterministic")
	}
	if !strings.HasPrefix(string(a), "openapi: 3.1") {
		t.Errorf("unexpected head: %.40s", a)
	}
}
