package jarvis

import (
	"testing"
	"time"

	"fleetdm/gm/pkg/ghapi"
)

func notif(reason, typ, url, title string) ghapi.Notification {
	n := ghapi.Notification{Reason: reason}
	n.Subject.Type = typ
	n.Subject.URL = url
	n.Subject.Title = title
	n.Repository.HTMLURL = "https://github.com/fleetdm/fleet"
	return n
}

func TestNotificationDedupAndClassify(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

	// A PR already on the board from a richer source.
	myPRs := []ghapi.PullRequest{
		{Number: 10, ReviewDecision: "APPROVED", Mergeable: "MERGEABLE", StatusCheckRollup: passingChecks(), UpdatedAt: now.Format(time.RFC3339)},
	}
	notifications := []ghapi.Notification{
		// Duplicate of PR #10 — should be skipped, not double-counted.
		notif("review_requested", "PullRequest", "https://api.github.com/repos/fleetdm/fleet/pulls/10", "dup"),
		// A mention on an issue not otherwise on the board — should surface.
		notif("mention", "Issue", "https://api.github.com/repos/fleetdm/fleet/issues/42", "mention me"),
	}

	board := BuildBoard("george", myPRs, nil, nil, nil, notifications, now)

	// PR #10 appears once (in quick wins), not duplicated.
	if got := len(board.Buckets[BucketQuickWins]); got != 1 {
		t.Errorf("quick wins = %d, want 1 (no dup)", got)
	}
	// The mention surfaces in waiting-on-you.
	found := false
	for _, it := range board.Buckets[BucketWaitingOnYou] {
		if it.Number == 42 && it.Kind == KindIssue {
			found = true
			if it.URL != "https://github.com/fleetdm/fleet/issues/42" {
				t.Errorf("bad html url: %s", it.URL)
			}
		}
	}
	if !found {
		t.Error("mention on issue #42 should surface in waiting-on-you")
	}
}
