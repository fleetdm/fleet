package main

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

// captureStdout provides scrumcheck behavior for this unit.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	_ = w.Close()

	var b bytes.Buffer
	_, _ = io.Copy(&b, r)
	return b.String()
}

// TestOutputHelpers provides scrumcheck behavior for this unit.
func TestOutputHelpers(t *testing.T) {
	t.Parallel()

	it := testIssueWithStatus(123, "Sample title", "https://github.com/fleetdm/fleet/issues/123", "Ready to estimate")
	v := DraftingCheckViolation{
		Item:      it,
		Unchecked: []string{"one", "two"},
		Status:    "Ready to estimate",
	}

	grouped := groupViolationsByStatus([]DraftingCheckViolation{v})
	if len(grouped["ready to estimate"]) != 1 {
		t.Fatalf("unexpected grouping: %#v", grouped)
	}

	out := captureStdout(t, func() {
		printDraftingStatusSection("Ready to estimate", []DraftingCheckViolation{v})
		printStaleAwaitingSummary(map[int][]StaleAwaitingViolation{
			71: {{
				Item:        it,
				StaleDays:   7,
				LastUpdated: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				ProjectNum:  71,
			}},
		}, 5)
		printTimestampCheckSummary(TimestampCheckResult{
			URL:          updatesTimestampURL,
			ExpiresAt:    time.Now().UTC().Add(10 * 24 * time.Hour),
			DurationLeft: 10 * 24 * time.Hour,
			MinDays:      5,
			OK:           true,
		})
		printTimestampCheckSummary(TimestampCheckResult{
			URL:   updatesTimestampURL,
			Error: "network",
		})
		printMissingMilestoneSummary([]MissingMilestoneIssue{{
			Item:       it,
			ProjectNum: 71,
			RepoOwner:  "fleetdm",
			RepoName:   "fleet",
			SuggestedMilestones: []MilestoneOption{
				{Number: 1, Title: "4.89.0"},
			},
		}})
	})

	for _, want := range []string{"Ready to estimate", "stale watchdog", "expires at", "Could not validate", "Missing milestone audit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

// TestGetBodyIssuePath provides scrumcheck behavior for this unit.
func TestGetBodyIssuePath(t *testing.T) {
	t.Parallel()

	var it Item
	u, _ := url.Parse("https://github.com/fleetdm/fleet/issues/999")
	it.Content.Issue.Number = githubv4.Int(999)
	it.Content.Issue.URL = githubv4.URI{URL: u}
	it.Content.Issue.Body = githubv4.String("Issue body")

	if got := getBody(it); got != "Issue body" {
		t.Fatalf("getBody(issue)=%q", got)
	}
}
