package main

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func inAwaitingQA(it Item) bool {
	for _, v := range it.FieldValues.Nodes {
		if string(v.SingleSelectValue.Name) == awaitingQAColumn {
			return true
		}
	}
	return false
}

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

// Remove leading emojis/symbols so we can match status names even if the project uses icons.
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

// Only flag if the unchecked checklist line exists.
// Ignore if missing or checked.
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

func shouldIgnoreDraftingChecklistItem(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, prefix := range draftingChecklistIgnorePrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

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

func isStaleAwaitingQA(it Item, now time.Time, staleAfter time.Duration) bool {
	if it.UpdatedAt.IsZero() {
		return false
	}
	return now.Sub(it.UpdatedAt.Time.UTC()) >= staleAfter
}
