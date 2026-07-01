package jarvis

import (
	"testing"
	"time"

	"fleetdm/gm/pkg/ghapi"
)

func TestTriageVisible(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	updated := now.Add(-1 * time.Hour)
	s := &TriageStore{Entries: map[string]TriageEntry{}}

	// Active by default.
	if !s.Visible("k", updated, now) {
		t.Fatal("unset key should be visible")
	}

	// Dismissed hides until the item changes.
	s.Dismiss("k", updated)
	if s.Visible("k", updated, now) {
		t.Error("dismissed item should be hidden")
	}
	if !s.Visible("k", updated.Add(time.Minute), now) {
		t.Error("dismissed item should resurface when updated")
	}

	// Snooze hides until the deadline.
	s.Snooze("k2", now.Add(time.Hour), updated)
	if s.Visible("k2", updated, now) {
		t.Error("snoozed item should be hidden before deadline")
	}
	if !s.Visible("k2", updated, now.Add(2*time.Hour)) {
		t.Error("snoozed item should reappear after deadline")
	}

	// Clear removes triage state.
	s.Clear("k")
	if !s.Visible("k", updated, now) {
		t.Error("cleared item should be visible")
	}
}

func TestLinkSessionsByBranch(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	myPRs := []ghapi.PullRequest{
		{Number: 10, HeadRefName: "gkarr-feature", ReviewDecision: "APPROVED", Mergeable: "MERGEABLE", StatusCheckRollup: passingChecks(), UpdatedAt: now.Format(time.RFC3339)},
	}
	sessions := []Session{
		{ID: "sess-1", Branch: "gkarr-feature", Title: "linked", LastActivity: now, WaitingOnMe: true},
		{ID: "sess-2", Branch: "orphan-branch", Title: "standalone", LastActivity: now, WaitingOnMe: true},
	}
	board := BuildBoard("george", myPRs, nil, nil, sessions, nil, now)

	// Linked session annotates the PR.
	var linked bool
	for _, it := range board.Buckets[BucketQuickWins] {
		if it.Number == 10 && it.HasSession && it.SessionID == "sess-1" {
			linked = true
		}
	}
	if !linked {
		t.Error("session sess-1 should be linked to PR #10")
	}
	// Unlinked session becomes a standalone sessions-bucket item.
	if len(board.Buckets[BucketSessions]) != 1 || board.Buckets[BucketSessions][0].SessionID != "sess-2" {
		t.Errorf("expected sess-2 standalone in sessions bucket, got %+v", board.Buckets[BucketSessions])
	}
}
