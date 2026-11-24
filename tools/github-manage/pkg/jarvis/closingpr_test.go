package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestPickClosingPR(t *testing.T) {
	ref := func(repo string, num int, state string) ghapi.ClosingPRRef {
		return ghapi.ClosingPRRef{Repo: repo, Number: num, State: state}
	}

	// Open beats merged, regardless of order.
	got := pickClosingPR([]ghapi.ClosingPRRef{
		ref("fleetdm/fleet", 1, "MERGED"),
		ref("fleetdm/fleet", 2, "OPEN"),
	})
	if got == nil || got.Number != 2 {
		t.Fatalf("expected open PR #2, got %+v", got)
	}

	// Merged is the fallback when nothing is open.
	got = pickClosingPR([]ghapi.ClosingPRRef{
		ref("fleetdm/fleet", 1, "CLOSED"),
		ref("fleetdm/confidential", 3, "MERGED"),
	})
	if got == nil || got.Number != 3 || got.Repo != "fleetdm/confidential" {
		t.Fatalf("expected merged PR #3, got %+v", got)
	}

	// Closed-unmerged only: nothing shipped, nothing to surface.
	if got = pickClosingPR([]ghapi.ClosingPRRef{ref("fleetdm/fleet", 1, "CLOSED")}); got != nil {
		t.Fatalf("expected nil for closed-only refs, got %+v", got)
	}
	if got = pickClosingPR(nil); got != nil {
		t.Fatalf("expected nil for empty refs, got %+v", got)
	}
}
