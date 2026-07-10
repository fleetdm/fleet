package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestNextAction(t *testing.T) {
	passingCI := []ghapi.StatusCheck{{Typename: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"}}
	mergeable := &ghapi.PullRequest{ReviewDecision: "APPROVED", Mergeable: "MERGEABLE", StatusCheckRollup: passingCI}
	blocked := &ghapi.PullRequest{ReviewDecision: "REVIEW_REQUIRED", Mergeable: "MERGEABLE", StatusCheckRollup: passingCI}

	tests := []struct {
		name string
		w    WorkItem
		want Action
	}{
		{"ready → start work", WorkItem{Status: "Ready"}, ActStartWork},
		{"unstarted → start work", WorkItem{Status: ""}, ActStartWork},
		{"in progress, session, no PR → open PR", WorkItem{Status: "In progress", SessionID: "s1"}, ActOpenPR},
		{"in progress, mergeable PR → mark in review", WorkItem{Status: "In progress", PR: &Item{PR: mergeable}}, ActMarkInReview},
		{"in review, mergeable PR → merge", WorkItem{Status: "In review", PR: &Item{PR: mergeable}}, ActMerge},
		{"in progress, blocked PR → address", WorkItem{Status: "In progress", PR: &Item{PR: blocked}}, ActAddressPR},
		{"in review, no open PR → awaiting QA", WorkItem{Status: "In review"}, ActMarkAwaitingQA},
		{"awaiting QA, nothing → none", WorkItem{Status: "Awaiting QA"}, ActNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.nextAction(); got != tt.want {
				t.Errorf("nextAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClosesIssues(t *testing.T) {
	pr := ghapi.PullRequest{Body: "This closes #38348 and also Fixes #100.\nresolved fleetdm/fleet#200"}
	got := pr.ClosesIssues()
	want := map[int]bool{38348: true, 100: true, 200: true}
	if len(got) != len(want) {
		t.Fatalf("ClosesIssues() = %v, want keys %v", got, want)
	}
	for _, n := range got {
		if !want[n] {
			t.Errorf("unexpected issue %d in %v", n, got)
		}
	}
}

func TestBuildWorkItemsLinksPRByClosingKeyword(t *testing.T) {
	b := Board{Buckets: map[Bucket][]Item{
		BucketNeedsYourHands: {{Kind: KindIssue, Number: 38348, Title: "do the thing", Issue: &ghapi.Issue{Number: 38348}}},
		BucketQuickWins: {{Kind: KindPR, Number: 55555, PR: &ghapi.PullRequest{
			Number: 55555, HeadRefName: "my-awesome-branch", Body: "Closes #38348",
			ReviewDecision: "APPROVED", Mergeable: "MERGEABLE",
			StatusCheckRollup: []ghapi.StatusCheck{{Typename: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"}},
		}}},
	}}
	links, _ := LoadLinkStore("")
	focus, _ := LoadFocusStore("")
	work := BuildWorkItems(b, links, focus, map[int]string{38348: "In progress"}, map[int]int{38348: 58})

	if len(work) != 1 {
		t.Fatalf("expected 1 work item, got %d", len(work))
	}
	w := work[0]
	if w.PR == nil || w.PR.Number != 55555 {
		t.Fatalf("expected PR #55555 linked, got %+v", w.PR)
	}
	if w.Branch != "my-awesome-branch" {
		t.Errorf("expected branch from PR head, got %q", w.Branch)
	}
	if w.Next != ActMarkInReview {
		t.Errorf("expected ActMarkInReview (in progress + mergeable), got %v", w.Next)
	}
}
