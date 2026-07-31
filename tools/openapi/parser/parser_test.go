package parser

import (
	"os"
	"testing"
)

func mustParseSample(t *testing.T) *Result {
	t.Helper()
	md, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	return Parse(string(md))
}

func TestParseFindsEndpoints(t *testing.T) {
	res := mustParseSample(t)
	if len(res.Endpoints) != 3 {
		t.Fatalf("want 3 endpoints, got %d", len(res.Endpoints))
	}
	e := res.Endpoints[0]
	if e.Section != "Widgets" || e.Heading != "List widgets" || e.Method != "GET" || e.Path != "/api/v1/fleet/widgets" {
		t.Errorf("unexpected first endpoint: %+v", e)
	}
	if e.Description != "Returns all widgets." {
		t.Errorf("unexpected description: %q", e.Description)
	}
}

func TestParseNormalizesPathParams(t *testing.T) {
	res := mustParseSample(t)
	if res.Endpoints[1].Path != "/api/v1/fleet/widgets/{id}" {
		t.Errorf("want normalized path, got %q", res.Endpoints[1].Path)
	}
}

func TestParseIgnoresBlockquotedRequestLines(t *testing.T) {
	res := mustParseSample(t)
	if res.Endpoints[1].Path == "/api/v1/fleet/old/widgets/{id}" {
		t.Error("picked up the deprecated blockquoted request line")
	}
}

func TestParseSkipsSectionsWithoutRequestLine(t *testing.T) {
	res := mustParseSample(t)
	if len(res.Skipped) != 1 || res.Skipped[0].Heading != "Just words" {
		t.Fatalf("want 1 skip for 'Just words', got %+v", res.Skipped)
	}
}
