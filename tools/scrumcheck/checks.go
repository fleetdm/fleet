package main

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// inAwaitingQA returns true when an item's status field is set to the
// configured Awaiting QA column name.
func inAwaitingQA(it Item) bool {
	for _, v := range it.FieldValues.Nodes {
		if string(v.SingleSelectValue.Name) == awaitingQAColumn {
			return true
		}
	}
	return false
}

// inDoneColumn returns true when an item's normalized status is "done".
func inDoneColumn(it Item) bool {
	for _, v := range it.FieldValues.Nodes {
		name := normalizeStatusName(string(v.SingleSelectValue.Name))
		if name == "done" {
			return true
		}
	}
	return false
}

// matchedStatus looks for any provided needle in the item's normalized status
// and returns the first matching needle.
func matchedStatus(it Item, needles []string) (string, bool) {
	for _, v := range it.FieldValues.Nodes {
		rawName := strings.TrimSpace(string(v.SingleSelectValue.Name))
		name := normalizeStatusName(rawName)
		for _, n := range needles {
			needle := strings.ToLower(strings.TrimSpace(n))
			if needle == "" {
				continue
			}
			if strings.Contains(name, needle) {
				return needle, true
			}
		}
	}
	return "", false
}

// normalizeStatusName strips leading emoji/symbol prefixes and lowercases the
// remaining status text so status comparisons are stable.
func normalizeStatusName(s string) string {
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			break
		}
		s = strings.TrimSpace(s[size:])
	}
	return strings.ToLower(s)
}

// hasUncheckedChecklistLine returns true only when the exact unchecked
// checklist item exists and no checked variant of that same text exists.
func hasUncheckedChecklistLine(body string, text string) bool {
	if body == "" || text == "" {
		return false
	}

	unchecked1 := "- [ ] " + text
	unchecked2 := "[ ] " + text

	checked := []string{
		"- [x] " + text,
		"- [X] " + text,
		"[x] " + text,
		"[X] " + text,
	}

	for _, c := range checked {
		if strings.Contains(body, c) {
			return false
		}
	}

	return strings.Contains(body, unchecked1) || strings.Contains(body, unchecked2)
}

// uncheckedChecklistItems extracts unchecked checklist entries from a body and
// filters out known drafting checklist lines we intentionally ignore.
func uncheckedChecklistItems(body string) []string {
	if body == "" {
		return nil
	}

	lines := strings.Split(body, "\n")
	out := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "- [ ] "):
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ] "))
			if !shouldIgnoreDraftingChecklistItem(text) {
				out = append(out, text)
			}
		case strings.HasPrefix(trimmed, "* [ ] "):
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "* [ ] "))
			if !shouldIgnoreDraftingChecklistItem(text) {
				out = append(out, text)
			}
		case strings.HasPrefix(trimmed, "[ ] "):
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "[ ] "))
			if !shouldIgnoreDraftingChecklistItem(text) {
				out = append(out, text)
			}
		}
	}
	return out
}

// shouldIgnoreDraftingChecklistItem reports whether a checklist line should be
// excluded from drafting violations based on configured ignore prefixes.
func shouldIgnoreDraftingChecklistItem(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, prefix := range draftingChecklistIgnorePrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

// uniqueInts removes duplicate integers while preserving first-seen order.
func uniqueInts(nums []int) []int {
	seen := make(map[int]bool, len(nums))
	out := make([]int, 0, len(nums))
	for _, n := range nums {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// normalizeLabelName trims whitespace, removes leading #/~ markers, and
// lowercases the label for consistent matching.
func normalizeLabelName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(s, "~")
	return strings.ToLower(strings.TrimSpace(s))
}

// compileLabelFilter normalizes CLI labels into a set used for fast lookups.
// Returns nil when no valid labels were provided.
func compileLabelFilter(labels []string) map[string]struct{} {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		norm := normalizeLabelName(label)
		if norm == "" {
			continue
		}
		out[norm] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// orderedGroupLabels returns normalized, de-duplicated labels in input order,
// excluding labels that begin with '-' (negative filters).
func orderedGroupLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, 0, len(labels))
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		if strings.HasPrefix(strings.TrimSpace(label), "-") {
			continue
		}
		norm := normalizeLabelName(label)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	return out
}

// matchesLabelFilter checks whether an item has at least one label from the
// filter set. A nil/empty filter matches all items.
func matchesLabelFilter(it Item, filter map[string]struct{}) bool {
	if len(filter) == 0 {
		return true
	}
	if it.Content.Issue.Number == 0 {
		return false
	}
	for _, n := range it.Content.Issue.Labels.Nodes {
		norm := normalizeLabelName(string(n.Name))
		if norm == "" {
			continue
		}
		if _, ok := filter[norm]; ok {
			return true
		}
	}
	return false
}

// isStaleAwaitingQA returns true when UpdatedAt is older than the stale window.
func isStaleAwaitingQA(it Item, now time.Time, staleAfter time.Duration) bool {
	if it.UpdatedAt.IsZero() {
		return false
	}
	return now.Sub(it.UpdatedAt.Time.UTC()) >= staleAfter
}
