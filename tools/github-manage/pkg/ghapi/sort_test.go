package ghapi

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
)

// helper to make an Issue with given number and label names
func mkIssue(num int, labels ...string) Issue {
	ls := make([]Label, 0, len(labels))
	for _, n := range labels {
		if n == "" {
			continue
		}
		ls = append(ls, Label{Name: n})
	}
	return Issue{Number: num, Labels: ls}
}

// local rank helpers (must mirror sort.go logic, but kept independent in tests)
func tPriorityRank(it Issue) int {
	rank := 3
	for _, l := range it.Labels {
		switch l.Name {
		case "P0":
			if rank > 0 {
				rank = 0
			}
		case "P1":
			if rank > 1 {
				rank = 1
			}
		case "P2":
			if rank > 2 {
				rank = 2
			}
		}
	}
	return rank
}

func tCustProspectRank(it Issue) int {
	for _, l := range it.Labels {
		n := strings.ToLower(l.Name)
		if strings.HasPrefix(n, "customer-") || strings.HasPrefix(n, "prospect-") {
			return 0
		}
	}
	return 1
}

func tTypeRank(it Issue) int {
	r := 3
	for _, l := range it.Labels {
		switch l.Name {
		case "story":
			if r > 0 {
				r = 0
			}
		case "bug":
			if r > 1 {
				r = 1
			}
		case "~sub-task":
			if r > 2 {
				r = 2
			}
		}
	}
	return r
}

func testLess(a, b Issue) bool {
	pa, pb := tPriorityRank(a), tPriorityRank(b)
	if pa != pb {
		return pa < pb
	}
	ca, cb := tCustProspectRank(a), tCustProspectRank(b)
	if ca != cb {
		return ca < cb
	}
	ta, tb := tTypeRank(a), tTypeRank(b)
	if ta != tb {
		return ta < tb
	}
	// number desc
	return a.Number > b.Number
}

func TestSortIssuesForDisplay_AllCombinations_ShuffleAndSort(t *testing.T) {
	priorities := [][]string{{"P0"}, {"P1"}, {"P2"}, {""}}
	custs := [][]string{{"customer-alpha"}, {""}}
	types := [][]string{{"story"}, {"bug"}, {"~sub-task"}, {"otherlabel"}}

	// generate all 32 combinations
	issues := make([]Issue, 0, len(priorities)*len(custs)*len(types))
	num := 1
	for _, p := range priorities {
		for _, c := range custs {
			for _, ty := range types {
				labels := append([]string{}, p...)
				labels = append(labels, c...)
				labels = append(labels, ty...)
				issues = append(issues, mkIssue(num, labels...))
				num++
			}
		}
	}

	// expected order via independent comparator
	expected := make([]Issue, len(issues))
	copy(expected, issues)
	sort.SliceStable(expected, func(i, j int) bool { return testLess(expected[i], expected[j]) })

	// shuffle original and sort using production
	shuffled := make([]Issue, len(issues))
	copy(shuffled, issues)
	r := rand.New(rand.NewSource(42))
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	SortIssuesForDisplay(shuffled)

	// compare number sequences
	for i := range expected {
		if expected[i].Number != shuffled[i].Number {
			t.Fatalf("mismatch at %d: expected #%d got #%d", i, expected[i].Number, shuffled[i].Number)
		}
	}
}

func TestSortIssuesForDisplay_TieBreaker_NumberDesc(t *testing.T) {
	a := mkIssue(10, "story")
	b := mkIssue(20, "story")
	// same ranks (no P*, no customer/prospect, same type), number desc => 20 then 10
	items := []Issue{a, b}
	SortIssuesForDisplay(items)
	if items[0].Number != 20 || items[1].Number != 10 {
		t.Fatalf("expected order [20,10], got [%d,%d]", items[0].Number, items[1].Number)
	}
}

func TestSortIssuesForDisplay_Stable(t *testing.T) {
	// identical ranks and numbers; ensure stability preserves original order
	a := mkIssue(100, "bug")
	b := mkIssue(100, "bug")
	// Embed an index via label to assert stability (since Issue contains slices and isn't comparable)
	a.Labels = append(a.Labels, Label{Name: "idx-a"})
	b.Labels = append(b.Labels, Label{Name: "idx-b"})
	items := []Issue{a, b}
	SortIssuesForDisplay(items)
	// when fully equal keys, order should be unchanged
	if items[0].Labels[len(items[0].Labels)-1].Name != "idx-a" || items[1].Labels[len(items[1].Labels)-1].Name != "idx-b" {
		t.Fatalf("expected stable order to be preserved")
	}
}
