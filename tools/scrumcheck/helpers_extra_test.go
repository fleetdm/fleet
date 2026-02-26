package main

import (
	"net/url"
	"strings"
	"testing"

	"github.com/shurcooL/githubv4"
)

// TestIssueAssigneesAndContainsLogin provides scrumcheck behavior for this unit.
func TestIssueAssigneesAndContainsLogin(t *testing.T) {
	t.Parallel()

	var it Item
	it.Content.Issue.Assignees.Nodes = []struct{ Login githubv4.String }{
		{Login: githubv4.String("Alice")},
		{Login: githubv4.String("alice")},
		{Login: githubv4.String("Bob")},
		{Login: githubv4.String("")},
	}
	got := issueAssignees(it)
	if len(got) != 2 {
		t.Fatalf("expected deduped assignees, got %v", got)
	}
	if !containsLogin(got, "alice") {
		t.Fatalf("expected contains alice in %v", got)
	}
	if containsLogin(got, "charlie") {
		t.Fatalf("unexpected charlie in %v", got)
	}
}

// TestItemStatusAndFileURLFromPath provides scrumcheck behavior for this unit.
func TestItemStatusAndFileURLFromPath(t *testing.T) {
	t.Parallel()

	it := testIssueWithStatus(200, "T", "https://github.com/fleetdm/fleet/issues/200", "In progress")
	if got := itemStatus(it); got != "In progress" {
		t.Fatalf("itemStatus=%q want In progress", got)
	}

	if got := fileURLFromPath("/tmp/report/index.html"); !strings.HasPrefix(got, "file:///tmp/report/index.html") {
		t.Fatalf("unexpected file URL: %q", got)
	}
}

// TestGetHelpersForIssueAndPR provides scrumcheck behavior for this unit.
func TestGetHelpersForIssueAndPR(t *testing.T) {
	t.Parallel()

	issue := testIssueWithStatus(300, "Issue title", "https://github.com/fleetdm/fleet/issues/300", "Ready")
	if got := getNumber(issue); got != 300 {
		t.Fatalf("issue number=%d want 300", got)
	}
	if got := getTitle(issue); got != "Issue title" {
		t.Fatalf("issue title=%q", got)
	}
	if got := getURL(issue); !strings.Contains(got, "/issues/300") {
		t.Fatalf("issue URL=%q", got)
	}

	var pr Item
	u, _ := url.Parse("https://github.com/fleetdm/fleet/pull/301")
	pr.Content.PullRequest.Number = githubv4.Int(301)
	pr.Content.PullRequest.Title = githubv4.String("PR title")
	pr.Content.PullRequest.Body = githubv4.String("PR body")
	pr.Content.PullRequest.URL = githubv4.URI{URL: u}
	if got := getNumber(pr); got != 301 {
		t.Fatalf("pr number=%d want 301", got)
	}
	if got := getTitle(pr); got != "PR title" {
		t.Fatalf("pr title=%q", got)
	}
	if got := getBody(pr); got != "PR body" {
		t.Fatalf("pr body=%q", got)
	}
	if got := getURL(pr); !strings.Contains(got, "/pull/301") {
		t.Fatalf("pr URL=%q", got)
	}
}
