package jarvis

import (
	"fmt"
	"sort"
	"time"

	"fleetdm/gm/pkg/ghapi"
)

// staleAfter is how long an assigned issue can go without an update before it
// drops from "needs your hands" to "cold".
const staleAfter = 14 * 24 * time.Hour

// parseTime parses a GitHub RFC3339 timestamp, returning the zero time on failure.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// BuildBoard classifies the user's PRs and assigned issues into leverage buckets.
//
//	myPRs     — open PRs the user authored
//	reviewPRs — open PRs requesting the user's review
//	issues    — open issues assigned to the user
//	login     — the user's GitHub login (to tell first-review from re-review)
//	now       — current time (injected for testability)
func BuildBoard(login string, myPRs, reviewPRs []ghapi.PullRequest, issues []ghapi.Issue, sessions []Session, notifications []ghapi.Notification, excludeIssues map[int]bool, now time.Time) Board {
	b := Board{Buckets: map[Bucket][]Item{}}
	add := func(it Item) { b.Buckets[it.Bucket] = append(b.Buckets[it.Bucket], it) }

	// Track which (kind, number) we've already placed so notifications only fill
	// gaps — things waiting on you that a richer source didn't already surface.
	present := map[string]bool{}
	pkey := func(k Kind, num int) string { return fmt.Sprintf("%d:%d", k, num) }

	for i := range myPRs {
		it := classifyMyPR(myPRs[i], login)
		add(it)
		present[pkey(KindPR, it.Number)] = true
	}
	for i := range reviewPRs {
		it := classifyReviewRequest(reviewPRs[i], login)
		add(it)
		present[pkey(KindPR, it.Number)] = true
	}
	for i := range issues {
		// Issues surfaced in the Project View are excluded here (and marked present
		// so a notification doesn't resurface them elsewhere).
		if excludeIssues[issues[i].Number] {
			present[pkey(KindIssue, issues[i].Number)] = true
			continue
		}
		it := classifyIssue(issues[i], now)
		add(it)
		present[pkey(KindIssue, it.Number)] = true
	}

	for _, n := range notifications {
		num := n.Number()
		if num == 0 {
			continue
		}
		kind := KindIssue
		if n.IsPR() {
			kind = KindPR
		}
		if present[pkey(kind, num)] {
			continue
		}
		bk, reason := classifyNotification(n)
		add(Item{
			Kind:             kind,
			Bucket:           bk,
			Number:           num,
			Title:            n.Subject.Title,
			URL:              n.HTMLURL(),
			Updated:          parseTime(n.UpdatedAt),
			Reason:           reason,
			FromNotification: true,
		})
		present[pkey(kind, num)] = true
	}

	linkSessions(&b, sessions)

	for bk := range b.Buckets {
		sortBucket(bk, b.Buckets[bk])
	}
	return b
}

// classifyNotification maps a notification's reason to a bucket and label.
func classifyNotification(n ghapi.Notification) (Bucket, string) {
	switch n.Reason {
	case "mention", "team_mention":
		return BucketWaitingOnYou, "mentioned you"
	case "author":
		return BucketWaitingOnYou, "new activity on your thread"
	case "review_requested":
		return BucketReviewQueue, "review requested"
	case "assign":
		return BucketNeedsYourHands, "assigned to you"
	case "comment":
		return BucketNeedsYourHands, "new comment"
	case "ci_activity":
		return BucketNeedsYourHands, "CI activity"
	default: // subscribed, state_change, manual, security_alert, ...
		return BucketCold, "activity (" + n.Reason + ")"
	}
}

// linkSessions attaches each waiting session to a PR with a matching branch, or
// surfaces it as a standalone item in the sessions bucket when unmatched. Only
// authored PRs carry headRefName (the review queue uses a lighter field set), so
// linking naturally targets your own work.
func linkSessions(b *Board, sessions []Session) {
	type loc struct {
		bk Bucket
		i  int
	}
	branchIdx := map[string]loc{}
	for _, bk := range BucketOrder {
		for i := range b.Buckets[bk] {
			it := b.Buckets[bk][i]
			if it.Kind == KindPR && it.PR != nil && it.PR.HeadRefName != "" {
				branchIdx[it.PR.HeadRefName] = loc{bk, i}
			}
		}
	}
	for _, s := range sessions {
		if s.Branch != "" {
			if l, ok := branchIdx[s.Branch]; ok {
				b.Buckets[l.bk][l.i].HasSession = true
				b.Buckets[l.bk][l.i].SessionID = s.ID
				b.Buckets[l.bk][l.i].Cwd = s.Cwd
				continue
			}
		}
		b.Buckets[BucketSessions] = append(b.Buckets[BucketSessions], Item{
			Kind:      KindSession,
			Bucket:    BucketSessions,
			Title:     s.Title,
			Updated:   s.LastActivity,
			Reason:    "waiting on your reply",
			SessionID: s.ID,
			Cwd:       s.Cwd,
			Branch:    s.Branch,
		})
	}
}

func baseItemFromPR(pr *ghapi.PullRequest) Item {
	return Item{
		Kind:    KindPR,
		Number:  pr.Number,
		Title:   pr.Title,
		URL:     pr.URL,
		Updated: parseTime(pr.UpdatedAt),
		PR:      pr,
	}
}

// classifyMyPR buckets a PR the user authored. Order matters: the first matching
// condition wins, so the most action-demanding state surfaces.
func classifyMyPR(pr ghapi.PullRequest, login string) Item {
	it := baseItemFromPR(&pr)
	switch {
	case pr.IsDraft:
		it.Bucket, it.Reason = BucketCold, "draft"
	case pr.ChangesRequested():
		it.Bucket, it.Reason = BucketWaitingOnYou, "changes requested — your move"
	case pr.UnresolvedThreads > 0:
		it.Bucket, it.Reason = BucketWaitingOnYou, fmt.Sprintf("%d unresolved review thread(s)", pr.UnresolvedThreads)
	case pr.HasOpenReviewerComment(login):
		it.Bucket, it.Reason = BucketWaitingOnYou, "reviewer left a comment"
	case pr.HasConflicts():
		it.Bucket, it.Reason = BucketNeedsYourHands, "merge conflicts"
	case pr.CIStatus() == "failing":
		it.Bucket, it.Reason = BucketNeedsYourHands, "CI failing"
	case pr.CanMergeNow():
		it.Bucket, it.Reason = BucketQuickWins, "✓CI ✓approved ✓no-conflicts"
	case pr.CIStatus() == "pending":
		it.Bucket, it.Reason = BucketCold, "CI running"
	default:
		it.Bucket, it.Reason = BucketCold, "awaiting review from others"
	}
	return it
}

// classifyReviewRequest buckets a PR that requests the user's review.
func classifyReviewRequest(pr ghapi.PullRequest, login string) Item {
	it := baseItemFromPR(&pr)
	if pr.ReviewedBy(login) {
		it.Bucket, it.Reason = BucketWaitingOnYou, "re-review (updated since you looked)"
	} else {
		it.Bucket, it.Reason = BucketReviewQueue, "awaiting your review"
	}
	return it
}

// classifyIssue buckets an assigned issue by staleness.
func classifyIssue(issue ghapi.Issue, now time.Time) Item {
	it := Item{
		Kind:    KindIssue,
		Number:  issue.Number,
		Title:   issue.Title,
		URL:     issue.URL,
		Updated: parseTime(issue.UpdatedAt),
		Issue:   &issue,
	}
	if !it.Updated.IsZero() && now.Sub(it.Updated) > staleAfter {
		it.Bucket, it.Reason = BucketCold, "assigned · stale"
	} else {
		it.Bucket, it.Reason = BucketNeedsYourHands, "assigned"
	}
	return it
}

// sortBucket orders items within a bucket. For buckets where the ball is in the
// user's court (waiting-on-you, review queue), oldest-first surfaces the most
// aged item. Everywhere else, most-recently-updated first.
func sortBucket(bk Bucket, items []Item) {
	switch bk {
	case BucketWaitingOnYou, BucketReviewQueue:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Updated.Before(items[j].Updated)
		})
	default:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Updated.After(items[j].Updated)
		})
	}
}
