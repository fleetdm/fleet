package jarvis

import (
	"testing"
	"time"

	"fleetdm/gm/pkg/ghapi"
)

func passingChecks() []ghapi.StatusCheck {
	return []ghapi.StatusCheck{{Typename: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"}}
}

func failingChecks() []ghapi.StatusCheck {
	return []ghapi.StatusCheck{{Typename: "CheckRun", Status: "COMPLETED", Conclusion: "FAILURE"}}
}

func TestCIStatus(t *testing.T) {
	tests := []struct {
		name   string
		checks []ghapi.StatusCheck
		want   string
	}{
		{"none", nil, "none"},
		{"passing", passingChecks(), "passing"},
		{"failing", failingChecks(), "failing"},
		{"pending checkrun", []ghapi.StatusCheck{{Typename: "CheckRun", Status: "IN_PROGRESS"}}, "pending"},
		{"failing wins over pending", []ghapi.StatusCheck{
			{Typename: "CheckRun", Status: "IN_PROGRESS"},
			{Typename: "CheckRun", Status: "COMPLETED", Conclusion: "FAILURE"},
		}, "failing"},
		{"status context failure", []ghapi.StatusCheck{{Typename: "StatusContext", State: "FAILURE"}}, "failing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := ghapi.PullRequest{StatusCheckRollup: tt.checks}
			if got := pr.CIStatus(); got != tt.want {
				t.Errorf("CIStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyMyPR(t *testing.T) {
	tests := []struct {
		name string
		pr   ghapi.PullRequest
		want Bucket
	}{
		{
			name: "mergeable now -> quick wins",
			pr:   ghapi.PullRequest{ReviewDecision: "APPROVED", Mergeable: "MERGEABLE", StatusCheckRollup: passingChecks()},
			want: BucketQuickWins,
		},
		{
			name: "changes requested -> waiting on you",
			pr:   ghapi.PullRequest{ReviewDecision: "CHANGES_REQUESTED", Mergeable: "MERGEABLE", StatusCheckRollup: passingChecks()},
			want: BucketWaitingOnYou,
		},
		{
			name: "CI failing -> needs your hands",
			pr:   ghapi.PullRequest{ReviewDecision: "REVIEW_REQUIRED", Mergeable: "MERGEABLE", StatusCheckRollup: failingChecks()},
			want: BucketNeedsYourHands,
		},
		{
			name: "conflicts -> needs your hands",
			pr:   ghapi.PullRequest{ReviewDecision: "APPROVED", Mergeable: "CONFLICTING", StatusCheckRollup: passingChecks()},
			want: BucketNeedsYourHands,
		},
		{
			name: "draft -> cold",
			pr:   ghapi.PullRequest{IsDraft: true, ReviewDecision: "APPROVED", Mergeable: "MERGEABLE", StatusCheckRollup: passingChecks()},
			want: BucketCold,
		},
		{
			name: "awaiting others -> cold",
			pr:   ghapi.PullRequest{ReviewDecision: "REVIEW_REQUIRED", Mergeable: "MERGEABLE", StatusCheckRollup: passingChecks()},
			want: BucketCold,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyMyPR(tt.pr, "george"); got.Bucket != tt.want {
				t.Errorf("classifyMyPR() bucket = %v (%q), want %v", got.Bucket, got.Reason, tt.want)
			}
		})
	}
}

func TestClassifyReviewRequest(t *testing.T) {
	notReviewed := ghapi.PullRequest{Number: 1}
	if got := classifyReviewRequest(notReviewed, "george"); got.Bucket != BucketReviewQueue {
		t.Errorf("unreviewed PR: got %v, want REVIEW_QUEUE", got.Bucket)
	}

	reReview := ghapi.PullRequest{
		Number:        2,
		LatestReviews: []ghapi.Review{{Author: ghapi.Author{Login: "george"}, State: "COMMENTED"}},
	}
	if got := classifyReviewRequest(reReview, "george"); got.Bucket != BucketWaitingOnYou {
		t.Errorf("re-review PR: got %v, want WAITING_ON_YOU", got.Bucket)
	}
}

func TestClassifyIssue(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	fresh := ghapi.Issue{Number: 1, UpdatedAt: now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)}
	if got := classifyIssue(fresh, now); got.Bucket != BucketNeedsYourHands {
		t.Errorf("fresh issue: got %v, want NEEDS_YOUR_HANDS", got.Bucket)
	}
	stale := ghapi.Issue{Number: 2, UpdatedAt: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)}
	if got := classifyIssue(stale, now); got.Bucket != BucketCold {
		t.Errorf("stale issue: got %v, want COLD", got.Bucket)
	}
}

func TestBuildBoardSorting(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	older := now.Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	newer := now.Add(-1 * 24 * time.Hour).Format(time.RFC3339)

	reviewPRs := []ghapi.PullRequest{
		{Number: 1, UpdatedAt: newer},
		{Number: 2, UpdatedAt: older},
	}
	board := BuildBoard("george", nil, reviewPRs, nil, nil, nil, nil, now)
	q := board.Buckets[BucketReviewQueue]
	if len(q) != 2 {
		t.Fatalf("expected 2 review items, got %d", len(q))
	}
	// Review queue sorts oldest-first to surface the most aged.
	if q[0].Number != 2 {
		t.Errorf("expected oldest (#2) first, got #%d", q[0].Number)
	}
}
