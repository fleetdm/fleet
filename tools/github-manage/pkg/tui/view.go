package tui

import (
	"fmt"
	"strings"

	"fleetdm/gm/pkg/ghapi"
)

func (m *model) generateIssueContent(issue ghapi.Issue) string {
	var content strings.Builder

	// Title
	content.WriteString(fmt.Sprintf("# %s\n\n", issue.Title))

	// Metadata section
	content.WriteString("## Issue Details\n\n")
	content.WriteString(fmt.Sprintf("**Number:** #%d\n\n", issue.Number))
	content.WriteString(fmt.Sprintf("**Type:** %s\n\n", issue.Typename))

	// Estimate
	if issue.Estimate > 0 {
		content.WriteString(fmt.Sprintf("**Estimate:** %d\n\n", issue.Estimate))
	} else {
		content.WriteString("**Estimate:** Not set\n\n")
	}

	// Labels
	if len(issue.Labels) > 0 {
		content.WriteString("**Labels:** ")
		var labelNames []string
		for _, label := range issue.Labels {
			labelNames = append(labelNames, label.Name)
		}
		content.WriteString(strings.Join(labelNames, ", "))
		content.WriteString("\n\n")
	} else {
		content.WriteString("**Labels:** None\n\n")
	}

	// State and metadata
	content.WriteString(fmt.Sprintf("**State:** %s\n\n", issue.State))
	content.WriteString(fmt.Sprintf("**Created:** %s\n\n", issue.CreatedAt))
	content.WriteString(fmt.Sprintf("**Updated:** %s\n\n", issue.UpdatedAt))

	// Author
	content.WriteString(fmt.Sprintf("**Author:** %s\n\n", issue.Author.Login))

	// Assignees
	if len(issue.Assignees) > 0 {
		content.WriteString("**Assignees:** ")
		var assigneeNames []string
		for _, assignee := range issue.Assignees {
			assigneeNames = append(assigneeNames, assignee.Login)
		}
		content.WriteString(strings.Join(assigneeNames, ", "))
		content.WriteString("\n\n")
	}

	// Milestone
	if issue.Milestone != nil {
		content.WriteString(fmt.Sprintf("**Milestone:** %s\n\n", issue.Milestone.Title))
	}

	// Description
	content.WriteString("## Description\n\n")
	if issue.Body != "" {
		content.WriteString(issue.Body)
	} else {
		content.WriteString("*No description provided.*")
	}
	content.WriteString("\n\n")

	return content.String()
}

func (m model) RenderLoading() string {
	var loadingMessage string
	switch m.commandType {
	// newview add if non default message is preferred when loading
	case IssuesCommand:
		loadingMessage = "Fetching Issues..."
	case ProjectCommand:
		loadingMessage = fmt.Sprintf("Fetching Project Items (ID: %d)...", m.projectID)
	case EstimatedCommand:
		loadingMessage = fmt.Sprintf("Fetching Estimated Tickets (Project: %d)...", m.projectID)
	default:
		loadingMessage = "Fetching Issues..."
	}
	return fmt.Sprintf("\n%s %s\n\n", m.spinner.View(), loadingMessage)
}

func (m model) RenderWorkflowRunning() string {
	s := "\n🚀 Workflow Execution in Progress\n\n"

	// Show overall progress
	s += fmt.Sprintf("Overall Progress: %s\n\n", m.overallProgress.View())

	// Determine window of tasks to display (show most recent 10, auto-scroll)
	totalTasks := len(m.tasks)
	lastFinished := -1
	for i := range m.tasks {
		if m.tasks[i].Status == TaskSuccess || m.tasks[i].Status == TaskError {
			lastFinished = i
		}
	}
	// Prefer to keep the currently running task in view
	lastIdx := lastFinished
	if m.currentTask > lastIdx {
		lastIdx = m.currentTask
	}
	if lastIdx < 0 {
		lastIdx = 0
	}
	windowSize := 10
	start := lastIdx - windowSize + 1
	if start < 0 {
		start = 0
	}
	end := start + windowSize
	if end > totalTasks {
		end = totalTasks
	}

	// Show individual task progress (windowed)
	if start > 0 {
		s += fmt.Sprintf("... %d earlier task(s) above ...\n", start)
	}
	for i := start; i < end; i++ {
		task := m.tasks[i]
		var statusIcon string
		var statusText string

		switch task.Status {
		case TaskPending:
			statusIcon = "⏳"
			statusText = pendingStyle.Render("PENDING")
		case TaskInProgress:
			statusIcon = "🔄"
			statusText = statusStyle.Render("RUNNING")
		case TaskSuccess:
			statusIcon = "✅"
			statusText = successStyle.Render("SUCCESS")
		case TaskError:
			statusIcon = "❌"
			statusText = errorStyle.Render("ERROR")
		}

		// Override icon for current task to show it's active
		if i == m.currentTask && task.Status != TaskSuccess && task.Status != TaskError {
			statusIcon = "🏃‍♀️"
		}

		s += fmt.Sprintf("%s %s %s", statusIcon, statusText, task.Description)

		if task.Error != nil {
			s += fmt.Sprintf(" - Error: %v", task.Error)
		}
		s += "\n"
	}
	if end < totalTasks {
		s += fmt.Sprintf("... %d more task(s) below ...\n", totalTasks-end)
	}

	// Add progress counter at the bottom
	completedTasks := 0
	for _, task := range m.tasks {
		if task.Status == TaskSuccess {
			completedTasks++
		}
		if task.Status == TaskError {
			completedTasks++
		}
	}

	progressPercent := 0.0
	if totalTasks > 0 {
		progressPercent = float64(completedTasks) / float64(totalTasks) * 100
	}

	s += fmt.Sprintf("\nProgress: %d/%d tasks completed (%.1f%%)", completedTasks, totalTasks, progressPercent)
	s += "\n\nPress 'q' to quit"
	return s
}

func (m model) RenderWorkflowComplete() string {
	s := "\n🎉 Workflow Complete!\n\n"

	// Show final progress (should be 100%)
	s += fmt.Sprintf("Final Progress: %s\n\n", m.overallProgress.View())

	// Show final task statuses
	successCount := 0
	errorCount := 0
	for _, task := range m.tasks {
		var statusIcon string
		var statusText string

		switch task.Status {
		case TaskSuccess:
			statusIcon = "✅"
			statusText = successStyle.Render("SUCCESS")
			successCount++
		case TaskError:
			statusIcon = "❌"
			statusText = errorStyle.Render("ERROR")
			errorCount++
		default:
			statusIcon = "⚠️"
			statusText = pendingStyle.Render("PENDING")
		}

		s += fmt.Sprintf("%s %s %s", statusIcon, statusText, task.Description)
		if task.Error != nil {
			s += fmt.Sprintf(" - Error: %v", task.Error)
		}
		s += "\n"
	}

	s += fmt.Sprintf("\nSummary: %d successful, %d failed out of %d tasks\n", successCount, errorCount, len(m.tasks))

	if m.errorMessage != "" {
		s += fmt.Sprintf("\n%s\n", m.errorMessage)
	}

	s += "\nPress any key to exit..."
	return s
}

func (m model) RenderIssueTable() string {
	s := ""
	currentChoices := m.getCurrentChoices()
	currentPos := m.cursor + 1
	totalFiltered := len(currentChoices)

	// Compute sum of estimates for currently selected issues
	sumSelectedEstimates := 0
	for idx := range m.selected {
		if idx >= 0 && idx < len(m.choices) {
			sumSelectedEstimates += m.choices[idx].Estimate
		}
	}

	// Header with filter info
	headerText := ""
	if m.filterInput != "" {
		headerText = fmt.Sprintf("GitHub Issues (%d/%d, Σest sel=%d) - Filtered by: '%s':\n\n", currentPos, totalFiltered, sumSelectedEstimates, m.filterInput)
	} else {
		headerText = fmt.Sprintf("GitHub Issues (%d/%d, Σest sel=%d):\n\n", currentPos, m.totalCount, sumSelectedEstimates)
	}

	warningBanner := ""
	if m.totalAvailable > 0 && m.totalAvailable > m.rawFetchedCount {
		missing := m.totalAvailable - m.rawFetchedCount
		warningBanner = errorStyle.Render(fmt.Sprintf("⚠ %d items not shown (limit=%d, total=%d). Increase --limit to include all issues.", missing, m.rawFetchedCount, m.totalAvailable)) + "\n\n"
	}

	s = warningBanner + headerText + fmt.Sprintf(" %-2d/%-2d Number Estimate  Type       Labels                              Title\n",
		m.selectedCount, m.totalCount)

	// Show indicator if there are issues above the visible area
	if m.viewOffset > 0 {
		s += fmt.Sprintf("  %s %-6s %s %s %s %s\n",
			" ", "...", "         ", "          ", strings.Repeat(" ", 35), "More issues above")
	}

	// Calculate which issues to display
	startIdx := m.viewOffset
	endIdx := m.viewOffset + m.viewHeight
	if endIdx > len(currentChoices) {
		endIdx = len(currentChoices)
	}

	// Display visible issues
	for i := startIdx; i < endIdx; i++ {
		issue := currentChoices[i]
		originalIndex := m.getOriginalIndex(i)
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		selected := ""
		if _, exists := m.selected[originalIndex]; exists {
			selected = "[x] "
		} else {
			selected = "[ ] "
		}
		s += fmt.Sprintf("%s %s %s %s %s %s %s\n",
			cursor, selected, fmt.Sprintf("%-6d", issue.Number), truncEstimate(issue.Estimate),
			truncType(issue.Typename), truncLables(issue.Labels), truncTitle(issue.Title))
	}

	// Show indicator if there are issues below the visible area
	if m.viewOffset+m.viewHeight < len(currentChoices) {
		s += fmt.Sprintf("  %s %-6s %s %s %s %s\n",
			" ", "...", "         ", "          ", strings.Repeat(" ", 35), "More issues below")
	}

	s += "\nNavigation: ↑/↓ or j/k (line), PgUp/PgDn (page), Home/End (top/bottom)\n"
	s += "Actions: enter/space/x (select), 'l' select all, 'h' deselect all, 'o' (view details), '/' (filter), 'w' (workflow), 'q' (quit)\n"
	return s
}

func (m model) RenderIssueDetail() string {
	s := ""
	if len(m.choices) > 0 && m.cursor < len(m.choices) {
		issue := m.choices[m.cursor]
		s += fmt.Sprintf("Issue #%d Details\n\n", issue.Number)
		s += m.detailViewport.View()
		s += "\n\nNavigation: ↑/↓ or j/k (line), PgUp/PgDn (page), Home/End (top/bottom)\n"
		s += "Actions: 'esc' (back to list), 'q' (quit)\n"
	} else {
		s = "No issue selected\n"
	}
	return s
}

func (m model) RenderFilterInput() string {
	currentChoices := m.getCurrentChoices()
	totalFiltered := len(currentChoices)

	s := fmt.Sprintf("Filter Issues - %d results:\n", totalFiltered)
	s += fmt.Sprintf("Filter: %s_\n\n", m.filterInput)

	// Show a preview of filtered results (top 10)
	previewCount := 10
	if totalFiltered < previewCount {
		previewCount = totalFiltered
	}

	s += "Preview:\n"
	for i := 0; i < previewCount; i++ {
		issue := currentChoices[i]
		originalIndex := m.getOriginalIndex(i)
		selected := ""
		if _, exists := m.selected[originalIndex]; exists {
			selected = "[x] "
		} else {
			selected = "[ ] "
		}
		s += fmt.Sprintf("  %s #%-6d %s\n", selected, issue.Number, truncTitle(issue.Title))
	}

	if totalFiltered > previewCount {
		s += fmt.Sprintf("  ... and %d more\n", totalFiltered-previewCount)
	}

	s += "\nType to filter by number, title, labels, or description\n"
	s += "Actions: 'enter' (apply filter), 'esc' (cancel), 'q' (quit)\n"
	return s
}

func (m model) RenderWorkflowSelection() string {
	s := "\n--- Workflow Selection ---\n"
	workflows := WorkflowTypeValues
	for i, workflow := range workflows {
		cursor := " "
		if i == m.workflowCursor {
			cursor = ">"
		}
		selected := "( )"
		if i == m.workflowCursor {
			selected = "(*)"
		}
		s += fmt.Sprintf("%s %s %s\n", cursor, selected, workflow)
	}
	s += "\nPress 'enter' to select, 'esc' to cancel.\n"
	return s
}

func (m model) RenderLabelInput() string {
	workflowName := "Add Label"
	if m.workflowType == BulkRemoveLabel {
		workflowName = "Remove Label"
	}
	s := fmt.Sprintf("\n--- %s ---\n", workflowName)
	s += fmt.Sprintf("Label: %s_\n", m.labelInput)
	s += "Press 'enter' to execute, 'esc' to cancel.\n"
	return s
}

func (m model) RenderProjectInput() string {
	workflowTitle := "Sprint Kickoff"
	promptText := "Target Project (ID or alias):"
	if m.workflowType == BulkKickOutOfSprint {
		workflowTitle = "Kick Out Of Sprint"
		promptText = "Source Project (ID or alias):"
	}
	s := fmt.Sprintf("\n--- %s ---\n", workflowTitle)
	s += fmt.Sprintf("%s %s_\n", promptText, m.projectInput)
	s += "Press 'enter' to execute, 'esc' to cancel.\n"
	return s
}
