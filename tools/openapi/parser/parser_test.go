package parser

import (
	"os"
	"strings"
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

func TestParseParamsTable(t *testing.T) {
	res := mustParseSample(t)
	e := res.Endpoints[0] // List widgets
	if len(e.Params) != 2 {
		t.Fatalf("want 2 params, got %+v", e.Params)
	}
	if e.Params[0].Name != "page" || e.Params[0].Type != "integer" || e.Params[0].In != "query" {
		t.Errorf("unexpected param: %+v", e.Params[0])
	}
}

func TestParseParamsNormalizesBodyAndRequired(t *testing.T) {
	res := mustParseSample(t)
	e := res.Endpoints[2] // Make widget
	if len(e.Params) != 2 {
		t.Fatalf("want 2 params, got %+v", e.Params)
	}
	name, count := e.Params[0], e.Params[1]
	if name.In != "body" || !name.Required {
		t.Errorf("json param not normalized to required body: %+v", name)
	}
	if count.In != "body" || count.Required {
		t.Errorf("body param wrong: %+v", count)
	}
	if !strings.Contains(name.Description, "The name.") {
		t.Errorf("description lost: %+v", name)
	}
}

func TestParsePathParamRequired(t *testing.T) {
	res := mustParseSample(t)
	e := res.Endpoints[1] // Get widget
	if len(e.Params) != 1 || e.Params[0].In != "path" || !e.Params[0].Required {
		t.Fatalf("want required path param, got %+v", e.Params)
	}
}

func TestParseRequiredVariants(t *testing.T) {
	cases := []struct {
		name   string
		desc   string
		wantRq bool
	}{
		{"bare Required", "**Required**. The widget ID.", true},
		{"Required with period inside bold", "**Required.** The widget ID.", true},
		{"lowercase required", "**required**. The widget ID.", true},
		{"conditional required is not required", "**Required if platform is Android**. The widget ID.", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := "## S\n\n### Get widget\n\n`GET /api/v1/fleet/widgets/:id`\n\n#### Parameters\n\n" +
				"| Name | Type    | In   | Description |\n" +
				"| ---- | ------- | ---- | ----------- |\n" +
				"| id   | integer | query | " + tc.desc + " |\n\n" +
				"##### Default response\n\n`Status: 200`\n\n```json\n{}\n```\n"
			res := Parse(md)
			if len(res.Endpoints) != 1 {
				t.Fatalf("want 1 endpoint, got %+v (skipped: %+v)", res.Endpoints, res.Skipped)
			}
			params := res.Endpoints[0].Params
			if len(params) != 1 {
				t.Fatalf("want 1 param, got %+v", params)
			}
			if params[0].Required != tc.wantRq {
				t.Errorf("Required = %v, want %v for description %q", params[0].Required, tc.wantRq, tc.desc)
			}
		})
	}
}

func TestParseDefaultResponse(t *testing.T) {
	res := mustParseSample(t)
	e := res.Endpoints[0] // List widgets
	if len(e.Responses) != 1 || e.Responses[0].Status != 200 {
		t.Fatalf("want one 200 response, got %+v", e.Responses)
	}
	obj, ok := e.Responses[0].Example.(map[string]any)
	if !ok {
		t.Fatalf("example not an object: %T", e.Responses[0].Example)
	}
	if _, ok := obj["widgets"]; !ok {
		t.Errorf("example missing widgets key: %v", obj)
	}
}

func TestParseInvalidJSONExampleIsError(t *testing.T) {
	md := "## S\n\n### Bad json\n\n`GET /api/v1/fleet/bad`\n\n##### Default response\n\n`Status: 200`\n\n```json\n{not json}\n```\n"
	res := Parse(md)
	if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0].Reason, "invalid JSON") {
		t.Fatalf("want invalid JSON skip, got %+v", res.Skipped)
	}
}

func TestStripLineComments(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "comment after value on same line is stripped",
			in:   "{\n  \"a\": 1, // comment\n  \"b\": 2\n}",
			want: "{\n  \"a\": 1, \n  \"b\": 2\n}",
		},
		{
			name: "url in string value survives byte-for-byte",
			in:   `{"url": "https://example.com/path"}`,
			want: `{"url": "https://example.com/path"}`,
		},
		{
			name: "escaped quote followed by // inside string is not stripped",
			in:   `{"a": "a\"b // not a comment"}`,
			want: `{"a": "a\"b // not a comment"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(stripLineComments([]byte(tc.in)))
			if got != tc.want {
				t.Errorf("stripLineComments(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestHardSkips(t *testing.T) {
	res := &Result{Skipped: []Skip{
		{Heading: "Retrieve your API token", Reason: ReasonNoRequestLine},
		{Heading: "Bad json", Reason: "invalid JSON in default response example: unexpected end of JSON input"},
	}}
	hard := HardSkips(res)
	if len(hard) != 1 || hard[0].Heading != "Bad json" {
		t.Fatalf("want only the non-request-line skip classified as hard, got %+v", hard)
	}
}

func TestParseDefaultResponseStripsLineComments(t *testing.T) {
	md := "## S\n\n### Commented json\n\n`GET /api/v1/fleet/commented`\n\n##### Default response\n\n`Status: 200`\n\n" +
		"```json\n{\n  \"count\": 1, // Fleet Premium only\n  \"url\": \"https://example.com/path\"\n}\n```\n"
	res := Parse(md)
	if len(res.Skipped) != 0 {
		t.Fatalf("want no skips, got %+v", res.Skipped)
	}
	if len(res.Endpoints) != 1 {
		t.Fatalf("want 1 endpoint, got %+v", res.Endpoints)
	}
	obj, ok := res.Endpoints[0].Responses[0].Example.(map[string]any)
	if !ok {
		t.Fatalf("example not an object: %T", res.Endpoints[0].Responses[0].Example)
	}
	if obj["count"] != float64(1) {
		t.Errorf("comment not stripped from value: %v", obj["count"])
	}
	if obj["url"] != "https://example.com/path" {
		t.Errorf("url mangled: %v", obj["url"])
	}
}
