package ghapi

import (
	"fmt"
	"sort"
	"strings"
)

// Handbook priority buckets, highest priority first, per
// handbook/company/product-groups.md#issue-prioritization.
const (
	bucketP0 = iota
	bucketP1
	bucketP2
	bucketCustomerPromise
	bucketActivationBlocker
	bucketReliability
	bucketBug
	bucketProductMaturity
	bucketOther
)

var priorityBucketNames = map[int]string{
	bucketP0:                "P0",
	bucketP1:                "P1",
	bucketP2:                "P2",
	bucketCustomerPromise:   "customer promise",
	bucketActivationBlocker: "activation blocker",
	bucketReliability:       "reliability",
	bucketBug:               "bug",
	bucketProductMaturity:   "product maturity",
	bucketOther:             "other",
}

func hasLabelIn(labels []string, name string) bool {
	for _, l := range labels {
		if strings.EqualFold(strings.TrimSpace(l), name) {
			return true
		}
	}
	return false
}

// HandbookPriorityBucket returns the handbook priority bucket for a set of
// label names. Lower is higher priority.
func HandbookPriorityBucket(labels []string) int {
	switch {
	case hasLabelIn(labels, "P0"):
		return bucketP0
	case hasLabelIn(labels, "P1"):
		return bucketP1
	case hasLabelIn(labels, "P2"):
		return bucketP2
	case hasLabelIn(labels, "~customer promise"):
		return bucketCustomerPromise
	case hasLabelIn(labels, "~activation-blocker"):
		return bucketActivationBlocker
	case hasLabelIn(labels, "reliability"):
		return bucketReliability
	case hasLabelIn(labels, "bug"):
		return bucketBug
	case hasLabelIn(labels, "~product-maturity"):
		return bucketProductMaturity
	default:
		return bucketOther
	}
}

// HandbookPriorityBucketName returns a human-readable name for the bucket an
// item's labels place it in.
func HandbookPriorityBucketName(labels []string) string {
	return priorityBucketNames[HandbookPriorityBucket(labels)]
}

// bugSubRank orders bugs within the bug bucket per
// handbook/company/product-groups.md#bug-prioritization: customer bugs, flaky
// tests, dogfood bugs, then the rest (oldest first via the number tiebreak in
// SortItemsByHandbookPriority).
func bugSubRank(labels []string) int {
	for _, l := range labels {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "customer-") {
			return 0
		}
	}
	for _, l := range labels {
		if strings.Contains(strings.ToLower(l), "flaky") {
			return 1
		}
	}
	if hasLabelIn(labels, "~dogfood") {
		return 2
	}
	return 3
}

type rankKey struct {
	bucket int
	sub    int
	number int
}

func makeRankKey(labels []string, number int) rankKey {
	return rankKey{bucket: HandbookPriorityBucket(labels), sub: bugSubRank(labels), number: number}
}

// rankLess orders by bucket; within the bug bucket by sub-rank then oldest
// first. Equal keys are ties (callers use stable sorts).
func rankLess(a, b rankKey) bool {
	if a.bucket != b.bucket {
		return a.bucket < b.bucket
	}
	if a.bucket == bucketBug {
		if a.sub != b.sub {
			return a.sub < b.sub
		}
		return a.number < b.number
	}
	return false
}

// SortItemsByHandbookPriority stable-sorts project items in place by the
// handbook issue prioritization of their own labels. Ties keep their current
// project order, except plain bugs which are ordered oldest first (lower
// issue number).
func SortItemsByHandbookPriority(items []ProjectItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return rankLess(makeRankKey(items[i].Labels, items[i].Content.Number),
			makeRankKey(items[j].Labels, items[j].Content.Number))
	})
}

// EffectiveRankLabels returns the labels to rank an item by: its parent
// issue's labels when the item is a sub-issue whose parent's labels are known
// (sub-issues inherit their parent's urgency), otherwise its own labels.
func EffectiveRankLabels(item ProjectItem, parents map[int]int, labelsByNumber map[int][]string) []string {
	if parentNum, ok := parents[item.Content.Number]; ok {
		if labels, ok := labelsByNumber[parentNum]; ok {
			return labels
		}
	}
	return item.Labels
}

// LabelsByNumber indexes fetched project items' labels by issue number, for
// use with EffectiveRankLabels.
func LabelsByNumber(items []ProjectItem) map[int][]string {
	m := make(map[int][]string, len(items))
	for _, it := range items {
		if it.Content.Number != 0 {
			m[it.Content.Number] = it.Labels
		}
	}
	return m
}

// DesiredColumnOrder returns a column's items in handbook priority order.
// Sub-issues (per parents, issue number -> parent issue number) whose parent
// is in the same column are placed directly below their parent, siblings
// keeping their current relative order. Sub-issues whose parent is elsewhere
// rank standalone but inherit the parent's labels when known (labelsByNumber
// should cover all fetched items, not just this column). parents may be nil.
func DesiredColumnOrder(column []ProjectItem, parents map[int]int, labelsByNumber map[int][]string) []ProjectItem {
	inColumn := make(map[int]bool, len(column))
	for _, it := range column {
		if it.Content.Number != 0 {
			inColumn[it.Content.Number] = true
		}
	}

	childrenOf := make(map[int][]ProjectItem)
	var topLevel []ProjectItem
	for _, it := range column {
		if parentNum, ok := parents[it.Content.Number]; ok && inColumn[parentNum] && parentNum != it.Content.Number {
			childrenOf[parentNum] = append(childrenOf[parentNum], it)
			continue
		}
		topLevel = append(topLevel, it)
	}

	sort.SliceStable(topLevel, func(i, j int) bool {
		li := EffectiveRankLabels(topLevel[i], parents, labelsByNumber)
		lj := EffectiveRankLabels(topLevel[j], parents, labelsByNumber)
		return rankLess(makeRankKey(li, topLevel[i].Content.Number),
			makeRankKey(lj, topLevel[j].Content.Number))
	})

	desired := make([]ProjectItem, 0, len(column))
	visited := make(map[int]bool, len(column))
	var emit func(it ProjectItem)
	emit = func(it ProjectItem) {
		if it.Content.Number != 0 {
			if visited[it.Content.Number] {
				return
			}
			visited[it.Content.Number] = true
		}
		desired = append(desired, it)
		for _, child := range childrenOf[it.Content.Number] {
			emit(child)
		}
	}
	for _, it := range topLevel {
		emit(it)
	}
	return desired
}

// RankMove is a single position mutation: place Item immediately after the
// item with AfterID, or at the top of the project when AfterID is empty.
type RankMove struct {
	Item    ProjectItem
	AfterID string
}

// PlanColumnRanking computes the position moves needed to rank each status
// column by handbook priority. items must be in current project order (the
// order gh project item-list returns). statuses optionally restricts which
// columns are ranked (case-insensitive substring match); empty means all.
// parents (issue number -> parent issue number, may be nil) makes sub-issues
// follow their parent per DesiredColumnOrder.
//
// A project has one global manual order and each board column renders its
// items in that order, so reordering a column's items relative to each other
// never disturbs another column. Items already in the right relative order
// (the longest increasing subsequence) are left alone, so re-running on a
// mostly-sorted project issues few mutations.
func PlanColumnRanking(items []ProjectItem, statuses []string, parents map[int]int) []RankMove {
	labelsByNumber := LabelsByNumber(items)
	byStatus := make(map[string][]ProjectItem)
	var statusOrder []string
	for _, it := range items {
		if !statusMatches(it.Status, statuses) {
			continue
		}
		if _, seen := byStatus[it.Status]; !seen {
			statusOrder = append(statusOrder, it.Status)
		}
		byStatus[it.Status] = append(byStatus[it.Status], it)
	}

	var moves []RankMove
	for _, status := range statusOrder {
		current := byStatus[status]
		desired := DesiredColumnOrder(current, parents, labelsByNumber)

		// Map each item to its desired index, then find the items already in
		// desired relative order within the current order.
		desiredIndex := make(map[string]int, len(desired))
		for i, it := range desired {
			desiredIndex[it.ID] = i
		}
		seq := make([]int, len(current))
		for i, it := range current {
			seq[i] = desiredIndex[it.ID]
		}
		keepDesiredIdx := longestIncreasingSubsequence(seq)

		prevID := ""
		for i, it := range desired {
			if !keepDesiredIdx[i] {
				moves = append(moves, RankMove{Item: it, AfterID: prevID})
			}
			prevID = it.ID
		}
	}
	return moves
}

func statusMatches(status string, statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, s := range statuses {
		if strings.Contains(strings.ToLower(status), strings.ToLower(strings.TrimSpace(s))) {
			return true
		}
	}
	return false
}

// longestIncreasingSubsequence returns the set of values (which are desired
// indices, a permutation of 0..n-1) that form a longest strictly increasing
// subsequence of seq, keyed by value.
func longestIncreasingSubsequence(seq []int) map[int]bool {
	n := len(seq)
	keep := make(map[int]bool, n)
	if n == 0 {
		return keep
	}
	length := make([]int, n) // LIS length ending at i
	parent := make([]int, n)
	best := 0
	for i := 0; i < n; i++ {
		length[i], parent[i] = 1, -1
		for j := 0; j < i; j++ {
			if seq[j] < seq[i] && length[j]+1 > length[i] {
				length[i] = length[j] + 1
				parent[i] = j
			}
		}
		if length[i] > length[best] {
			best = i
		}
	}
	for i := best; i >= 0; i = parent[i] {
		keep[seq[i]] = true
	}
	return keep
}

// UpdateProjectItemPosition moves a project item in the project's global
// manual order, placing it immediately after afterID (or at the very top when
// afterID is empty). Board views render each status column in this order.
// Views with an explicit sort applied ignore manual positions.
func UpdateProjectItemPosition(projectID int, itemID, afterID string) error {
	projectNodeID, err := getProjectNodeID(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project node ID: %v", err)
	}
	after := ""
	if afterID != "" {
		after = fmt.Sprintf(`, afterId: "%s"`, afterID)
	}
	command := fmt.Sprintf(`gh api graphql -f query='mutation { updateProjectV2ItemPosition(input: { projectId: "%s", itemId: "%s"%s }) { clientMutationId } }'`,
		projectNodeID, itemID, after)
	if _, err := RunCommandWithRetry(command, 3); err != nil {
		return fmt.Errorf("failed to update item position: %v", err)
	}
	return nil
}

// RankStatusColumnsByPriority ranks each status column of a project by the
// handbook priority metric, applying the minimal position mutations needed.
// items must be in current project order. parents may be nil to skip
// sub-issue handling. progress, if non-nil, is called after each applied
// move. Returns the number of items moved.
func RankStatusColumnsByPriority(projectID int, items []ProjectItem, statuses []string, parents map[int]int, progress func(done, total int, move RankMove)) (int, error) {
	moves := PlanColumnRanking(items, statuses, parents)
	for i, m := range moves {
		if err := UpdateProjectItemPosition(projectID, m.Item.ID, m.AfterID); err != nil {
			return i, fmt.Errorf("moving #%d (%s): %v", m.Item.Content.Number, m.Item.Title, err)
		}
		if progress != nil {
			progress(i+1, len(moves), m)
		}
	}
	return len(moves), nil
}
