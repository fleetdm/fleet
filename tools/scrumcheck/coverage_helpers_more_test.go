package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

func TestChecksHelpersBranches(t *testing.T) {
	it := testIssueWithStatus(123, "status item", "https://github.com/fleetdm/fleet/issues/123", "✔️Awaiting QA")
	it.Content.Issue.Labels.Nodes = append(it.Content.Issue.Labels.Nodes, struct{ Name githubv4.String }{Name: "#G-Orchestration"})

	if _, ok := matchedStatus(it, []string{"", "awaiting qa"}); !ok {
		t.Fatal("expected matched status")
	}
	if _, ok := matchedStatus(it, []string{"in progress"}); ok {
		t.Fatal("did not expect status match")
	}

	filter := compileLabelFilter([]string{"#g-orchestration"})
	if !matchesLabelFilter(it, filter) {
		t.Fatal("expected label filter match")
	}
	it.Content.Issue.Number = 0
	if matchesLabelFilter(it, filter) {
		t.Fatal("expected no match when issue number is zero")
	}

	if normalizeStatusName("  ✅ In Review ") != "in review" {
		t.Fatal("normalizeStatusName did not strip symbols correctly")
	}

	now := time.Now().UTC()
	it.UpdatedAt = githubv4.DateTime{Time: now.Add(-49 * time.Hour)}
	if !isStaleAwaitingQA(it, now, 48*time.Hour) {
		t.Fatal("expected stale awaiting QA item")
	}
}

func TestPreviewBodyLinesAndHasLabelBranches(t *testing.T) {
	if out := previewBodyLines("a\nb", 0); out != nil {
		t.Fatalf("expected nil when maxLines <= 0, got %#v", out)
	}
	out := previewBodyLines(" \nline1\n\nline2\n", 2)
	if len(out) != 2 || out[0] != "line1" || out[1] != "line2" {
		t.Fatalf("unexpected preview lines: %#v", out)
	}

	if !hasLabel([]string{"#g-orchestration", "bug"}, "g-orchestration") {
		t.Fatal("expected normalized label match")
	}
	if hasLabel([]string{"bug"}, "security") {
		t.Fatal("did not expect non-matching label")
	}
}

func TestParseBridgeOpSignalBranches(t *testing.T) {
	if _, ok := parseBridgeOpSignal("plain text"); ok {
		t.Fatal("expected parse failure for non-BRIDGE_OP input")
	}
	if _, ok := parseBridgeOpSignal("BRIDGE_OP op=apply stage="); ok {
		t.Fatal("expected parse failure when required fields are empty")
	}

	evt, ok := parseBridgeOpSignal("BRIDGE_OP op=apply stage=done item=42 elapsed=bad")
	if !ok {
		t.Fatal("expected parse success")
	}
	if evt.Issue != "42" {
		t.Fatalf("expected item alias to map into issue, got %q", evt.Issue)
	}
	if evt.Status != "working" || evt.Repo != "item" || evt.Caller != "unknown" {
		t.Fatalf("expected defaulted fields, got %+v", evt)
	}
}

func TestAssigneeFetchersErrorBranches(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/user"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("{bad-json")),
			}, nil
		case strings.Contains(r.URL.Path, "/assignees"):
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		case strings.Contains(r.URL.Path, "/search/issues"):
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("forbidden")),
			}, nil
		default:
			return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
		}
	}))

	if got := fetchViewerLogin(context.Background(), "tok"); got != "" {
		t.Fatalf("expected empty login on decode error, got %q", got)
	}
	if got := fetchRepoAssignees(context.Background(), "tok", "fleetdm", "fleet"); len(got) != 0 {
		t.Fatalf("expected no assignees on non-200, got %#v", got)
	}
	if got := fetchAssignedIssuesByProject(context.Background(), "tok", "fleetdm", 97); len(got) != 0 {
		t.Fatalf("expected no assigned issues on failed search, got %#v", got)
	}
}
