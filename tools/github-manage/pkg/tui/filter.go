package tui

import (
	"fmt"
	"strings"

	"fleetdm/gm/pkg/ghapi"
)

func (m *model) applyFilter() {
	if m.filterInput == "" {
		// No filter, show all issues
		// Ensure base list is sorted
		ghapi.SortIssuesForDisplay(m.choices)
		m.filteredChoices = m.choices
		m.originalIndices = make([]int, len(m.choices))
		for i := range m.originalIndices {
			m.originalIndices[i] = i
		}
		return
	}

	filter := strings.ToLower(m.filterInput)
	m.filteredChoices = nil
	m.originalIndices = nil

	// Ensure base list is sorted before applying filter
	ghapi.SortIssuesForDisplay(m.choices)
	for i, issue := range m.choices {
		if m.matchesFilter(issue, filter) {
			m.filteredChoices = append(m.filteredChoices, issue)
			m.originalIndices = append(m.originalIndices, i)
		}
	}

	// Reset cursor if it's beyond the filtered results
	if m.cursor >= len(m.filteredChoices) {
		m.cursor = 0
	}
}

// --- Sorting helpers for Normal view ordering ---

// labelNamesLower returns a set of lowercase label names for fast lookup.
// Sorting moved to ghapi.SortIssuesForDisplay

func (m *model) matchesFilter(issue ghapi.Issue, filter string) bool {
	// Check issue number
	if strings.Contains(strings.ToLower(fmt.Sprintf("#%d", issue.Number)), filter) {
		return true
	}
	if strings.Contains(strings.ToLower(fmt.Sprintf("%d", issue.Number)), filter) {
		return true
	}

	// Check title
	if strings.Contains(strings.ToLower(issue.Title), filter) {
		return true
	}

	// Check body/description
	if strings.Contains(strings.ToLower(issue.Body), filter) {
		return true
	}

	// Check labels
	for _, label := range issue.Labels {
		if strings.Contains(strings.ToLower(label.Name), filter) {
			return true
		}
	}

	// Check type
	if strings.Contains(strings.ToLower(issue.Typename), filter) {
		return true
	}

	// Check author
	if strings.Contains(strings.ToLower(issue.Author.Login), filter) {
		return true
	}

	// Check assignees
	for _, assignee := range issue.Assignees {
		if strings.Contains(strings.ToLower(assignee.Login), filter) {
			return true
		}
	}

	return false
}

func (m *model) getCurrentChoices() []ghapi.Issue {
	if m.filterInput != "" {
		return m.filteredChoices
	}
	return m.choices
}

func (m *model) getOriginalIndex(filteredIndex int) int {
	if m.filterInput != "" && filteredIndex < len(m.originalIndices) {
		return m.originalIndices[filteredIndex]
	}
	return filteredIndex
}
