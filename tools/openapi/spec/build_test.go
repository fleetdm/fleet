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

func TestBuildEmitsAllParsedOperations(t *testing.T) {
	doc, err := Build(sampleResult())
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
	post, ok := item["post"]
	if !ok {
		t.Fatal("post op missing")
	}
	if post.RequestBody == nil {
		t.Fatal("post has no request body")
	}
	schema := post.RequestBody.Content["application/json"].Schema
	req := schema["required"].([]string)
	if len(req) != 1 || req[0] != "name" {
		t.Errorf("required wrong: %v", req)
	}
}

func TestBuildEmitsEveryEndpointRegardlessOfCount(t *testing.T) {
	res := sampleResult()
	res.Endpoints = append(res.Endpoints, parser.Endpoint{
		Section: "Gadgets", Heading: "List gadgets", Method: "GET",
		Path: "/api/v1/fleet/gadgets",
		Responses: []parser.Response{
			{Status: 200, Example: map[string]any{"gadgets": []any{}}},
		},
	})
	doc, err := Build(res)
	if err != nil {
		t.Fatal(err)
	}

	var gotOps int
	for _, methods := range doc.Paths {
		gotOps += len(methods)
	}
	if gotOps != len(res.Endpoints) {
		t.Fatalf("want all %d parsed endpoints emitted, got %d operations", len(res.Endpoints), gotOps)
	}
	if _, ok := doc.Paths["/api/v1/fleet/gadgets"]["get"]; !ok {
		t.Errorf("newly added endpoint not emitted: %v", doc.Paths)
	}
}

func TestBuildFailsOnDuplicateOperationID(t *testing.T) {
	res := sampleResult()
	res.Endpoints = append(res.Endpoints, parser.Endpoint{
		Section: "Widgets", Heading: "List widgets", Method: "GET",
		Path: "/api/v1/fleet/other-widgets",
	})
	_, err := Build(res)
	if err == nil || !strings.Contains(err.Error(), "duplicate operationId") {
		t.Fatalf("want duplicate operationId error, got %v", err)
	}
}

func TestRenderDeterministic(t *testing.T) {
	doc, err := Build(sampleResult())
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

func TestBuildFailsOnDuplicateMethodPath(t *testing.T) {
	res := sampleResult()
	dup := res.Endpoints[0]
	dup.Heading = "List widgets again"
	res.Endpoints = append(res.Endpoints, dup)
	_, err := Build(res)
	if err == nil || !strings.Contains(err.Error(), "duplicate endpoint GET /api/v1/fleet/widgets") {
		t.Fatalf("want duplicate endpoint error, got %v", err)
	}
}
