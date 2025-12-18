package util

import "strings"

// HasLabel returns true if labels contains a label equal to want (case-insensitive, trimmed).
func HasLabel(labels []string, want string) bool {
	lw := strings.ToLower(strings.TrimSpace(want))
	for _, l := range labels {
		if strings.ToLower(strings.TrimSpace(l)) == lw {
			return true
		}
	}
	return false
}

// HasAnyLabel returns true if labels contains any of the wants.
func HasAnyLabel(labels []string, wants ...string) bool {
	for _, w := range wants {
		if HasLabel(labels, w) {
			return true
		}
	}
	return false
}

// HasLabelPrefix returns true if any label starts with the given prefix (case-insensitive).
func HasLabelPrefix(labels []string, prefix string) bool {
	lp := strings.ToLower(strings.TrimSpace(prefix))
	for _, l := range labels {
		ll := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(ll, lp) {
			return true
		}
	}
	return false
}
