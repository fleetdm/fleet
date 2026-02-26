package main

import (
	"net/url"
	"testing"

	"github.com/shurcooL/githubv4"
)

func TestBuildBridgePolicy(t *testing.T) {
	t.Parallel()

	drafting := []DraftingCheckViolation{
		{
			Item:      testIssueItem(40007, "https://github.com/fleetdm/fleet/issues/40007"),
			Unchecked: []string{"check one", "check two"},
		},
	}
	missing := []MissingMilestoneIssue{
		{
			Item:      testIssueItem(40007, "https://github.com/fleetdm/fleet/issues/40007"),
			RepoOwner: "fleetdm",
			RepoName:  "fleet",
			SuggestedMilestones: []MilestoneOption{
				{Number: 101, Title: "Sprint 1"},
				{Number: 102, Title: "Sprint 2"},
			},
		},
	}

	p := buildBridgePolicy(drafting, missing)
	key := issueKey("fleetdm/fleet", 40007)

	if !p.ChecklistByIssue[key]["check one"] || !p.ChecklistByIssue[key]["check two"] {
		t.Fatalf("expected checklist allowlist entries for %s", key)
	}
	if !p.MilestonesByIssue[key][101] || !p.MilestonesByIssue[key][102] {
		t.Fatalf("expected milestone allowlist entries for %s", key)
	}
}

func TestBridgeAllowlistChecks(t *testing.T) {
	t.Parallel()

	b := &uiBridge{
		allowChecklist: map[string]map[string]bool{
			issueKey("fleetdm/fleet", 123): {"allowed checklist item": true},
		},
		allowMilestones: map[string]map[int]bool{
			issueKey("fleetdm/fleet", 123): {55: true},
		},
	}

	if !b.isAllowedChecklist("fleetdm/fleet", 123, "allowed checklist item") {
		t.Fatal("expected checklist to be allowed")
	}
	if b.isAllowedChecklist("fleetdm/fleet", 123, "something else") {
		t.Fatal("unexpected checklist allow")
	}
	if !b.isAllowedMilestone("fleetdm/fleet", 123, 55) {
		t.Fatal("expected milestone to be allowed")
	}
	if b.isAllowedMilestone("fleetdm/fleet", 123, 99) {
		t.Fatal("unexpected milestone allow")
	}
}

func testIssueItem(num int, issueURL string) Item {
	var it Item
	it.Content.Issue.Number = githubv4.Int(num)
	u, _ := url.Parse(issueURL)
	it.Content.Issue.URL = githubv4.URI{URL: u}
	return it
}
