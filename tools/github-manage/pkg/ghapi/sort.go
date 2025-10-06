package ghapi

import (
	"sort"
	"strings"
)

// labelNamesLower returns a set of lowercase label names for fast lookup.
func labelNamesLower(issue Issue) map[string]struct{} {
	names := make(map[string]struct{}, len(issue.Labels))
	for _, l := range issue.Labels {
		names[strings.ToLower(strings.TrimSpace(l.Name))] = struct{}{}
	}
	return names
}

// hasAnyLabelPrefix checks if any label starts with given prefixes (case-insensitive).
func hasAnyLabelPrefix(issue Issue, prefixes ...string) bool {
	for _, l := range issue.Labels {
		ln := strings.ToLower(strings.TrimSpace(l.Name))
		for _, p := range prefixes {
			if strings.HasPrefix(ln, strings.ToLower(p)) {
				return true
			}
		}
	}
	return false
}

// priorityRank returns 0 for P0, 1 for P1, 2 for P2, 3 for none.
func priorityRank(issue Issue) int {
	rank := 3
	for _, l := range issue.Labels {
		ln := strings.ToUpper(strings.TrimSpace(l.Name))
		switch ln {
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

// custProspectRank returns 0 if any label starts with customer- or prospect-, else 1.
func custProspectRank(issue Issue) int {
	if hasAnyLabelPrefix(issue, "customer-", "prospect-") {
		return 0
	}
	return 1
}

// typeRank returns 0 for story, 1 for bug, 2 for ~sub-task, 3 otherwise.
func typeRank(issue Issue) int {
	names := labelNamesLower(issue)
	if _, ok := names["story"]; ok {
		return 0
	}
	if _, ok := names["bug"]; ok {
		return 1
	}
	if _, ok := names["~sub-task"]; ok {
		return 2
	}
	return 3
}

// SortIssuesForDisplay sorts issues in-place using the following precedence:
// 1) Priority labels: P0, P1, P2, then none
// 2) Presence of labels starting with customer- or prospect-
// 3) Type labels: story, bug, ~sub-task, then others
// 4) Issue number descending
func SortIssuesForDisplay(items []Issue) {
	sort.SliceStable(items, func(i, j int) bool {
		// 1) Priority P0/P1/P2/none
		pi, pj := priorityRank(items[i]), priorityRank(items[j])
		if pi != pj {
			return pi < pj
		}
		// 2) Customer/Prospect present first
		ci, cj := custProspectRank(items[i]), custProspectRank(items[j])
		if ci != cj {
			return ci < cj
		}
		// 3) Type story, bug, ~sub-task, others
		ti, tj := typeRank(items[i]), typeRank(items[j])
		if ti != tj {
			return ti < tj
		}
		// 4) Number descending
		return items[i].Number > items[j].Number
	})
}
