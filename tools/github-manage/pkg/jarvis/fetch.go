package jarvis

import (
	"time"

	"fleetdm/gm/pkg/ghapi"
)

// FetchResult is the data backing one render of the dashboard.
type FetchResult struct {
	Login string
	Board Board
}

// Fetch gathers the user's PRs, review requests, and assigned issues from the
// repo and classifies them into the leverage board. It runs the gh calls
// sequentially; each one wraps the gh CLI via ghapi.
func Fetch(repo string, limit int) (FetchResult, error) {
	login, err := ghapi.GetCurrentLogin()
	if err != nil {
		return FetchResult{}, err
	}
	myPRs, err := ghapi.GetMyPullRequests(repo, limit)
	if err != nil {
		return FetchResult{}, err
	}
	// Enrich your PRs with unresolved review-thread counts (a "your move" signal
	// reviewDecision misses). Per-PR GraphQL; best-effort per PR.
	for i := range myPRs {
		if c, err := ghapi.GetUnresolvedReviewThreadCount(repo, myPRs[i].Number); err == nil {
			myPRs[i].UnresolvedThreads = c
		}
	}
	reviewPRs, err := ghapi.GetReviewRequestedPRs(repo, limit)
	if err != nil {
		return FetchResult{}, err
	}
	issues, err := ghapi.GetAssignedIssues(repo, limit)
	if err != nil {
		return FetchResult{}, err
	}
	// These sources are best-effort — a failure shouldn't blank the dashboard.
	// (Notifications need the gh `notifications` scope; cherry-picks need a local
	// RC ref; sessions need ~/.claude/projects.)
	notifications, _ := ghapi.GetNotifications(repo)
	sessions, _ := DiscoverSessions(30)

	board := BuildBoard(login, myPRs, reviewPRs, issues, sessions, notifications, time.Now())
	board.AddItems(PendingCherryPicks(repo))
	return FetchResult{Login: login, Board: board}, nil
}
