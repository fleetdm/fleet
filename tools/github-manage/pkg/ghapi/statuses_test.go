package ghapi

import (
	"errors"
	"testing"
)

func TestParseStatusChunk(t *testing.T) {
	body := []byte(`{"data":{"repository":{
		"i0":{"projectItems":{"nodes":[
			{"project":{"number":108,"title":"🍎 #g-apple-at-work","updatedAt":"2026-09-01T00:00:00Z"},
			 "fieldValueByName":{"name":"In progress"}},
			{"project":{"number":97,"title":"Drafting","updatedAt":"2026-08-01T00:00:00Z"},
			 "fieldValueByName":null}
		]}},
		"i1":null
	}}}`)

	got, ok := parseStatusChunk(body, nil, []int{100, 200})
	if !ok {
		t.Fatal("expected ok for a well-formed response")
	}
	if len(got) != 1 {
		t.Fatalf("expected only i0 to parse (i1 is null), got %v", got)
	}
	found := got[100]
	if ps := found[108]; !ps.Present || ps.Status != "In progress" || ps.Title != "🍎 #g-apple-at-work" {
		t.Errorf("project 108 = %+v", ps)
	}
	if ps := found[97]; !ps.Present || ps.Status != "" {
		t.Errorf("project 97 (unset Status) = %+v", ps)
	}
	if _, exists := got[200]; exists {
		t.Error("null alias should mean the issue is absent, not empty")
	}
}

func TestParseStatusChunkPartialErrorKeepsData(t *testing.T) {
	// gh exits non-zero on a partial GraphQL error (one alias NOT_FOUND) but
	// still prints the response; trailing stderr noise must not break parsing.
	body := []byte(`{"data":{"repository":{
		"i0":{"projectItems":{"nodes":[]}},
		"i1":null
	}},"errors":[{"type":"NOT_FOUND"}]}
gh: Could not resolve to an Issue`)
	got, ok := parseStatusChunk(body, errors.New("exit status 1"), []int{1, 2})
	if !ok {
		t.Fatal("partial errors with data should still be usable")
	}
	if _, exists := got[1]; !exists {
		t.Error("issue 1 (present, no projects) should be in the result")
	}
}

func TestParseStatusChunkFailure(t *testing.T) {
	if _, ok := parseStatusChunk([]byte("gh: HTTP 502 Bad Gateway"), errors.New("exit status 1"), []int{1}); ok {
		t.Error("non-JSON output must report not-ok so the caller falls back")
	}
	if _, ok := parseStatusChunk([]byte(`{"data":{"repository":{}}}`), errors.New("exit status 1"), []int{1}); ok {
		t.Error("errored request with no data must report not-ok")
	}
}
