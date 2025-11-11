package tui

import (
	"fleetdm/gm/pkg/ghapi"
	"fleetdm/gm/pkg/logger"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) HandleHotkeys(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.workflowState {
		case Loading:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		case WorkflowRunning:
			// Only allow quitting during workflow execution
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		case WorkflowComplete:
			// Exit when workflow is complete
			switch msg.String() {
			case "q", "ctrl+c", "enter", "esc", " ":
				return m, tea.Quit
			}
			return m, nil
		case NormalMode:
			switch msg.String() {
			case "/":
				m.workflowState = FilterInput
				m.filterInput = ""
				m.applyFilter()
			case "w":
				if len(m.selected) > 0 {
					m.workflowState = WorkflowSelection
					m.workflowCursor = 0
					m.errorMessage = ""
				}
			case "o":
				currentChoices := m.getCurrentChoices()
				if len(currentChoices) > 0 && m.cursor < len(currentChoices) {
					// Get the original issue (not filtered)
					originalIndex := m.getOriginalIndex(m.cursor)
					if originalIndex < len(m.choices) {
						issue := m.choices[originalIndex]
						content := m.generateIssueContent(issue)

						if m.glamourRenderer != nil {
							rendered, err := m.glamourRenderer.Render(content)
							if err != nil {
								// Fallback to plain content if rendering fails
								m.issueContent = content
							} else {
								m.issueContent = rendered
							}
						} else {
							m.issueContent = content
						}

						m.detailViewport.SetContent(m.issueContent)
						m.workflowState = IssueDetail
					}
				}
			case "j", "down":
				currentChoices := m.getCurrentChoices()
				if m.cursor < len(currentChoices)-1 {
					m.cursor++
					m.adjustViewForCursor()
				}
			case "k", "up":
				if m.cursor > 0 {
					m.cursor--
					m.adjustViewForCursor()
				}
			case "pgdown", "ctrl+f":
				// Page down - move cursor by view height
				currentChoices := m.getCurrentChoices()
				newCursor := m.cursor + m.viewHeight
				if newCursor >= len(currentChoices) {
					newCursor = len(currentChoices) - 1
				}
				m.cursor = newCursor
				m.adjustViewForCursor()
			case "pgup", "ctrl+b":
				// Page up - move cursor by view height
				newCursor := m.cursor - m.viewHeight
				if newCursor < 0 {
					newCursor = 0
				}
				m.cursor = newCursor
				m.adjustViewForCursor()
			case "home", "ctrl+a":
				// Go to first issue
				m.cursor = 0
				m.adjustViewForCursor()
			case "end", "ctrl+e":
				// Go to last issue
				currentChoices := m.getCurrentChoices()
				m.cursor = len(currentChoices) - 1
				m.adjustViewForCursor()
			case "enter", "x", " ":
				currentChoices := m.getCurrentChoices()
				if m.cursor < len(currentChoices) {
					originalIndex := m.getOriginalIndex(m.cursor)
					if _, exists := m.selected[originalIndex]; exists {
						delete(m.selected, originalIndex)
					} else {
						m.selected[originalIndex] = struct{}{}
					}
					m.selectedCount = len(m.selected)
				}
			case "s":
				// Toggle selection of related issues (parent/sub-tasks)
				currentChoices := m.getCurrentChoices()
				if m.cursor < len(currentChoices) {
					issue := currentChoices[m.cursor]
					// Load from cache or fetch
					related, ok := m.relatedCache[issue.Number]
					if !ok {
						if rel, err := ghapi.GetRelatedIssueNumbers(issue.Number); err == nil {
							related = rel
							m.relatedCache[issue.Number] = related
						}
					}
					related = append(related, issue.Number)
					logger.Debugf("Related issues for #%d: %v", issue.Number, related)
					if len(related) > 0 {
						// Determine if all related currently selected
						allSelected := true
						for _, rn := range related {
							// find index for issue number in m.choices
							for idx, iss := range m.choices {
								if iss.Number == rn {
									if _, ok := m.selected[idx]; !ok {
										allSelected = false
									}
									break
								}
							}
						}
						// Toggle accordingly
						for _, rn := range related {
							for idx, iss := range m.choices {
								if iss.Number == rn {
									if allSelected {
										delete(m.selected, idx)
									} else {
										m.selected[idx] = struct{}{}
									}
									break
								}
							}
						}
						m.selectedCount = len(m.selected)
					}
				}
			case "l":
				// Select all visible (filtered) issues
				m.selected = make(map[int]struct{})
				currentChoices := m.getCurrentChoices()
				for i := range currentChoices {
					orig := m.getOriginalIndex(i)
					m.selected[orig] = struct{}{}
				}
				m.selectedCount = len(m.selected)
			case "h":
				// Clear all selections
				m.selected = make(map[int]struct{})
				m.selectedCount = 0
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case IssueDetail:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.workflowState = NormalMode
			case "j", "down":
				m.detailViewport.LineDown(1)
			case "k", "up":
				m.detailViewport.LineUp(1)
			case "pgdown", "ctrl+f":
				m.detailViewport.HalfViewDown()
			case "pgup", "ctrl+b":
				m.detailViewport.HalfViewUp()
			case "home", "ctrl+a":
				m.detailViewport.GotoTop()
			case "end", "ctrl+e":
				m.detailViewport.GotoBottom()
			}
		case FilterInput:
			switch msg.String() {
			case "esc":
				m.workflowState = NormalMode
				m.filterInput = ""
				m.applyFilter()
				m.cursor = 0
				m.adjustViewForCursor()
			case "enter":
				m.workflowState = NormalMode
				m.adjustViewForCursor()
			case "backspace":
				if len(m.filterInput) > 0 {
					m.filterInput = m.filterInput[:len(m.filterInput)-1]
					m.applyFilter()
					m.cursor = 0
					m.adjustViewForCursor()
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			default:
				// Add character to filter
				if len(msg.String()) == 1 {
					m.filterInput += msg.String()
					m.applyFilter()
					m.cursor = 0
					m.adjustViewForCursor()
				}
			}
		case WorkflowSelection:
			switch msg.String() {
			case "j", "down":
				if m.workflowCursor < len(WorkflowTypeValues)-1 {
					m.workflowCursor++
				}
			case "k", "up":
				if m.workflowCursor > 0 {
					m.workflowCursor--
				}
			case "enter":
				m.workflowType = WorkflowType(m.workflowCursor)
				switch m.workflowType {
				// newworkflow Add to switch to support
				case BulkAddLabel, BulkRemoveLabel:
					m.workflowState = LabelInput
					m.labelInput = ""
				case BulkDemoSummary:
					return m, m.executeWorkflow()
				case BulkSprintKickoff, BulkKickOutOfSprint:
					if m.projectID != 0 {
						// Use the provided project ID
						return m, m.executeWorkflow()
					} else {
						m.workflowState = ProjectInput
						m.projectInput = ""
					}
				case BulkMilestoneClose:
					return m, m.executeWorkflow()
				}
			case "esc":
				m.workflowState = NormalMode
				m.errorMessage = ""
			}
		case LabelInput:
			switch msg.String() {
			case "enter":
				if m.labelInput != "" {
					return m, m.executeWorkflow()
				}
			case "esc":
				m.workflowState = NormalMode
				m.errorMessage = ""
			case "backspace":
				if len(m.labelInput) > 0 {
					m.labelInput = m.labelInput[:len(m.labelInput)-1]
				}
			default:
				// Add character to input
				if len(msg.String()) == 1 {
					m.labelInput += msg.String()
				}
			}
		case ProjectInput:
			switch msg.String() {
			case "enter":
				if m.projectInput != "" {
					return m, m.executeWorkflow()
				}
			case "esc":
				m.workflowState = NormalMode
				m.errorMessage = ""
			case "backspace":
				if len(m.projectInput) > 0 {
					m.projectInput = m.projectInput[:len(m.projectInput)-1]
				}
			default:
				// Add character to input
				if len(msg.String()) == 1 {
					m.projectInput += msg.String()
				}
			}
		}
	default:
		// Unknown state, just return
		return m, nil
	}
	return m, nil
}
