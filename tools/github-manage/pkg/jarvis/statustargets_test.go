package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func mkNotification(repo, subjectType, subjectURL string) ghapi.Notification {
	var n ghapi.Notification
	n.Repository.FullName = repo
	n.Subject.Type = subjectType
	n.Subject.URL = subjectURL
	return n
}

func TestStatusTargets(t *testing.T) {
	issues := []ghapi.Issue{
		{Number: 100, URL: "https://github.com/fleetdm/fleet/issues/100"},
		{Number: 200, URL: "https://github.com/fleetdm/confidential/issues/200"},
	}
	notifications := []ghapi.Notification{
		// Issue notification in another repo: included with its own repo.
		mkNotification("fleetdm/confidential", "Issue", "https://api.github.com/repos/fleetdm/confidential/issues/40693"),
		// PR notification: excluded.
		mkNotification("fleetdm/fleet", "PullRequest", "https://api.github.com/repos/fleetdm/fleet/pulls/300"),
		// Duplicate of an assigned issue: deduped.
		mkNotification("fleetdm/fleet", "Issue", "https://api.github.com/repos/fleetdm/fleet/issues/100"),
		// No number (release notification): excluded.
		mkNotification("fleetdm/fleet", "Release", "https://api.github.com/repos/fleetdm/fleet/releases/tag"),
	}

	got := statusTargets(issues, notifications, "fleetdm/fleet")
	want := []issueStatusTarget{
		{repo: "fleetdm/fleet", number: 100},
		{repo: "fleetdm/confidential", number: 200},
		{repo: "fleetdm/confidential", number: 40693},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d targets, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("target %d = %+v, want %+v", i, got[i], w)
		}
	}
}
