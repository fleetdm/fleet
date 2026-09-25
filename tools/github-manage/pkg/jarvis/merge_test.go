package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

// TestMergeTargetFromProjectViewIssue verifies that merging works when the cursor
// is on a Project View issue (KindIssue) whose PR isn't separately selectable: the
// PR and the issue/project (for the Awaiting QA follow-up) must resolve.
func TestMergeTargetFromProjectViewIssue(t *testing.T) {
	prItem := &Item{Kind: KindPR, Number: 49380, PR: &ghapi.PullRequest{Number: 49380}}
	m := &Model{
		flat:   []Item{{Kind: KindIssue, Number: 49364}},
		cursor: 0,
		work:   []WorkItem{{Number: 49364, Project: 108, PR: prItem}},
		workByIssue: map[int]WorkItem{
			49364: {Number: 49364, Project: 108, PR: prItem},
		},
	}

	got, issue, project, ok := m.mergeTarget()
	if !ok {
		t.Fatal("expected mergeTarget to resolve a PR for the selected issue")
	}
	if got.Number != 49380 {
		t.Errorf("PR number = %d, want 49380", got.Number)
	}
	if issue != 49364 || project != 108 {
		t.Errorf("issue/project = %d/%d, want 49364/108 (needed for Awaiting QA)", issue, project)
	}
}

// TestMergeTargetIssueWithoutPR: an issue with no linked PR has nothing to merge.
func TestMergeTargetIssueWithoutPR(t *testing.T) {
	m := &Model{
		flat:        []Item{{Kind: KindIssue, Number: 49364}},
		cursor:      0,
		workByIssue: map[int]WorkItem{49364: {Number: 49364, Project: 108}},
	}
	if _, _, _, ok := m.mergeTarget(); ok {
		t.Error("expected no merge target for an issue without a PR")
	}
}
