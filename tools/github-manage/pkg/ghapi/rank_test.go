package ghapi

import (
	"testing"
)

func mkItem(id string, num int, status string, labels ...string) ProjectItem {
	return ProjectItem{
		ID:      id,
		Title:   id,
		Status:  status,
		Labels:  labels,
		Content: ProjectItemContent{Number: num},
	}
}

func TestHandbookPriorityBucketOrder(t *testing.T) {
	cases := []struct {
		labels []string
		want   int
	}{
		{[]string{"P0", "bug"}, bucketP0},
		{[]string{"p1"}, bucketP1},
		{[]string{"P2", "~customer promise"}, bucketP2},
		{[]string{"~customer promise", "story"}, bucketCustomerPromise},
		{[]string{"~activation-blocker"}, bucketActivationBlocker},
		{[]string{"reliability", "bug"}, bucketReliability},
		{[]string{"bug", "customer-acme"}, bucketBug},
		{[]string{"~product-maturity", "story"}, bucketProductMaturity},
		{[]string{"story"}, bucketOther},
		{nil, bucketOther},
	}
	for _, c := range cases {
		if got := HandbookPriorityBucket(c.labels); got != c.want {
			t.Errorf("HandbookPriorityBucket(%v) = %d, want %d", c.labels, got, c.want)
		}
	}
}

func TestSortItemsByHandbookPriority(t *testing.T) {
	items := []ProjectItem{
		mkItem("story", 10, "", "story"),
		mkItem("old-bug", 5, "", "bug"),
		mkItem("new-bug", 20, "", "bug"),
		mkItem("dogfood-bug", 30, "", "bug", "~dogfood"),
		mkItem("flaky-bug", 40, "", "bug", "~flaky test"),
		mkItem("customer-bug", 50, "", "bug", "customer-acme"),
		mkItem("maturity", 60, "", "story", "~product-maturity"),
		mkItem("reliability", 70, "", "bug", "reliability"),
		mkItem("blocker", 80, "", "~activation-blocker"),
		mkItem("promise", 90, "", "story", "~customer promise"),
		mkItem("p2", 100, "", "P2", "bug"),
		mkItem("p1", 110, "", "P1"),
		mkItem("p0", 120, "", "P0"),
	}
	SortItemsByHandbookPriority(items)

	want := []string{
		"p0", "p1", "p2",
		"promise", "blocker", "reliability",
		"customer-bug", "flaky-bug", "dogfood-bug",
		"old-bug", "new-bug", // plain bugs oldest first
		"maturity", "story",
	}
	for i, w := range want {
		if items[i].ID != w {
			t.Fatalf("position %d = %s, want %s", i, items[i].ID, w)
		}
	}
}

func TestSortItemsByHandbookPriorityStableTies(t *testing.T) {
	items := []ProjectItem{
		mkItem("a", 3, "", "story"),
		mkItem("b", 1, "", "story"),
		mkItem("c", 2, "", "story"),
	}
	SortItemsByHandbookPriority(items)
	for i, w := range []string{"a", "b", "c"} {
		if items[i].ID != w {
			t.Fatalf("tie order changed: position %d = %s, want %s", i, items[i].ID, w)
		}
	}
}

func TestPlanColumnRankingAlreadySorted(t *testing.T) {
	items := []ProjectItem{
		mkItem("a", 1, "Ready", "P1"),
		mkItem("b", 2, "Ready", "bug"),
		mkItem("c", 3, "Ready", "story"),
	}
	if moves := PlanColumnRanking(items, nil, nil); len(moves) != 0 {
		t.Fatalf("expected no moves for sorted column, got %d", len(moves))
	}
}

func TestPlanColumnRankingMinimalMoves(t *testing.T) {
	// Only "p0" is out of place; the rest are already in relative order.
	items := []ProjectItem{
		mkItem("bug", 1, "Ready", "bug"),
		mkItem("story", 2, "Ready", "story"),
		mkItem("p0", 3, "Ready", "P0"),
	}
	moves := PlanColumnRanking(items, nil, nil)
	if len(moves) != 1 {
		t.Fatalf("expected 1 move, got %d: %+v", len(moves), moves)
	}
	if moves[0].Item.ID != "p0" || moves[0].AfterID != "" {
		t.Fatalf("expected p0 moved to top, got item %s after %q", moves[0].Item.ID, moves[0].AfterID)
	}
}

func TestPlanColumnRankingAnchors(t *testing.T) {
	// Reversed column: desired order is c, b, a. LIS keeps one item; the other
	// two chain after their desired predecessor.
	items := []ProjectItem{
		mkItem("a", 1, "Ready", "story"),
		mkItem("b", 2, "Ready", "bug"),
		mkItem("c", 3, "Ready", "P0"),
	}
	moves := PlanColumnRanking(items, nil, nil)
	if len(moves) != 2 {
		t.Fatalf("expected 2 moves, got %d: %+v", len(moves), moves)
	}
	if moves[0].Item.ID != "c" || moves[0].AfterID != "" {
		t.Fatalf("first move should put c on top, got %+v", moves[0])
	}
	if moves[1].Item.ID != "b" || moves[1].AfterID != "c" {
		t.Fatalf("second move should put b after c, got %+v", moves[1])
	}
}

func TestPlanColumnRankingColumnsIndependent(t *testing.T) {
	items := []ProjectItem{
		mkItem("r-bug", 1, "Ready", "bug"),
		mkItem("ip-p1", 2, "In progress", "P1"),
		mkItem("r-p0", 3, "Ready", "P0"),
		mkItem("ip-story", 4, "In progress", "story"),
	}
	moves := PlanColumnRanking(items, nil, nil)
	// Ready needs r-p0 on top; In progress is already ordered (P1 before story).
	if len(moves) != 1 || moves[0].Item.ID != "r-p0" {
		t.Fatalf("expected only r-p0 to move, got %+v", moves)
	}
}

func TestPlanColumnRankingStatusFilter(t *testing.T) {
	items := []ProjectItem{
		mkItem("r-bug", 1, "Ready", "bug"),
		mkItem("r-p0", 2, "Ready", "P0"),
		mkItem("d-bug", 3, "Done", "bug"),
		mkItem("d-p0", 4, "Done", "P0"),
	}
	moves := PlanColumnRanking(items, []string{"ready"}, nil)
	if len(moves) != 1 || moves[0].Item.ID != "r-p0" {
		t.Fatalf("expected only Ready column ranked, got %+v", moves)
	}
}

func TestDesiredColumnOrderSubIssuesFollowParent(t *testing.T) {
	// Sub-issues 11 and 12 belong to P1 story 10; they carry only
	// ~sub-task/~frontend labels but must land right below their parent, in
	// their current relative order, ahead of the standalone bug.
	column := []ProjectItem{
		mkItem("sub-fe", 11, "Ready", "~sub-task", "~frontend"),
		mkItem("bug", 20, "Ready", "bug"),
		mkItem("story-p1", 10, "Ready", "story", "P1"),
		mkItem("sub-be", 12, "Ready", "~sub-task", "~backend"),
	}
	parents := map[int]int{11: 10, 12: 10}
	desired := DesiredColumnOrder(column, parents, LabelsByNumber(column))
	want := []string{"story-p1", "sub-fe", "sub-be", "bug"}
	for i, w := range want {
		if desired[i].ID != w {
			t.Fatalf("position %d = %s, want %s (full: %v)", i, desired[i].ID, w, ids(desired))
		}
	}
}

func TestDesiredColumnOrderSubIssueInheritsRemoteParentPriority(t *testing.T) {
	// Parent P1 story 10 is in another column; its sub-issue still outranks
	// the plain bug in this column by inheriting the parent's labels.
	all := []ProjectItem{
		mkItem("story-p1", 10, "In progress", "story", "P1"),
		mkItem("bug", 20, "Ready", "bug"),
		mkItem("sub", 11, "Ready", "~sub-task"),
	}
	parents := map[int]int{11: 10}
	column := all[1:]
	desired := DesiredColumnOrder(column, parents, LabelsByNumber(all))
	want := []string{"sub", "bug"}
	for i, w := range want {
		if desired[i].ID != w {
			t.Fatalf("position %d = %s, want %s (full: %v)", i, desired[i].ID, w, ids(desired))
		}
	}
}

func TestDesiredColumnOrderUnknownParentFallsBackToOwnLabels(t *testing.T) {
	column := []ProjectItem{
		mkItem("sub", 11, "Ready", "~sub-task"),
		mkItem("bug", 20, "Ready", "bug"),
	}
	// Parent 99 is not among fetched items, so sub ranks by its own labels
	// (other bucket) below the bug.
	parents := map[int]int{11: 99}
	desired := DesiredColumnOrder(column, parents, LabelsByNumber(column))
	want := []string{"bug", "sub"}
	for i, w := range want {
		if desired[i].ID != w {
			t.Fatalf("position %d = %s, want %s (full: %v)", i, desired[i].ID, w, ids(desired))
		}
	}
}

func TestPlanColumnRankingWithParents(t *testing.T) {
	// Already in desired order (parent, its subs, then bug): no moves.
	items := []ProjectItem{
		mkItem("story-p1", 10, "Ready", "story", "P1"),
		mkItem("sub-fe", 11, "Ready", "~sub-task", "~frontend"),
		mkItem("bug", 20, "Ready", "bug"),
	}
	parents := map[int]int{11: 10}
	if moves := PlanColumnRanking(items, nil, parents); len(moves) != 0 {
		t.Fatalf("expected no moves, got %+v", moves)
	}
	// Sub drifted to the bottom: one move, anchored after its parent.
	items = []ProjectItem{
		mkItem("story-p1", 10, "Ready", "story", "P1"),
		mkItem("bug", 20, "Ready", "bug"),
		mkItem("sub-fe", 11, "Ready", "~sub-task", "~frontend"),
	}
	moves := PlanColumnRanking(items, nil, parents)
	if len(moves) != 1 || moves[0].Item.ID != "sub-fe" || moves[0].AfterID != "story-p1" {
		t.Fatalf("expected sub-fe moved after story-p1, got %+v", moves)
	}
}

func ids(items []ProjectItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

func TestLongestIncreasingSubsequence(t *testing.T) {
	keep := longestIncreasingSubsequence([]int{2, 0, 1, 3})
	// LIS is 0,1,3.
	for _, v := range []int{0, 1, 3} {
		if !keep[v] {
			t.Errorf("expected %d in LIS keep set", v)
		}
	}
	if keep[2] {
		t.Errorf("did not expect 2 in LIS keep set")
	}
	if len(longestIncreasingSubsequence(nil)) != 0 {
		t.Errorf("expected empty keep set for empty input")
	}
}
