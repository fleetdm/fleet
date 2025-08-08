package main

import (
	"fmt"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type CommandType int

const (
	IssuesCommand CommandType = iota
	ProjectCommand
	EstimatedCommand
)

type WorkflowState int

const (
	Loading WorkflowState = iota
	NormalMode
	IssueDetail
	FilterInput
	WorkflowSelection
	LabelInput
	ProjectInput
	WorkflowRunning
	WorkflowComplete
)

type WorkflowType int

const (
	BulkAddLabel WorkflowType = iota
	BulkRemoveLabel
	BulkSprintKickoff
	BulkMilestoneClose
	BulkKickOutOfSprint
)

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskInProgress
	TaskSuccess
	TaskError
)

type WorkflowTask struct {
	ID          int
	Description string
	Status      TaskStatus
	Progress    float64
	Error       error
}

var (
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#04B575")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF0000")).
			Padding(0, 1)

	pendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#888888")).
			Padding(0, 1)
)

type processTaskMsg struct {
	taskID int
}

type actionStatusMsg struct {
	index int
	state string
}

type startAsyncWorkflowMsg struct {
	actions []ghapi.Action
}

type asyncStatusMsg struct {
	status ghapi.Status
}

type model struct {
	choices       []ghapi.Issue
	cursor        int
	selected      map[int]struct{}
	spinner       spinner.Model
	totalCount    int
	selectedCount int
	// Scrolling support
	viewOffset int // Offset for scrolling view
	viewHeight int // Number of visible lines for issues
	// Filtering support
	filterInput     string
	filteredChoices []ghapi.Issue
	originalIndices []int // Maps filtered index to original index
	// Issue detail view
	detailViewport  viewport.Model
	glamourRenderer *glamour.TermRenderer
	issueContent    string
	// Command-specific parameters
	commandType CommandType
	projectID   int
	limit       int
	search      string
	// Workflow state
	workflowState  WorkflowState
	workflowType   WorkflowType
	workflowCursor int
	labelInput     string
	projectInput   string
	errorMessage   string
	// Progress tracking
	tasks              []WorkflowTask
	currentTask        int
	overallProgress    progress.Model
	githubOpInProgress bool              // Ensure only one GitHub operation at a time
	currentActions     []ghapi.Action    // Store current actions being processed
	statusChan         chan ghapi.Status // Channel for receiving status updates from AsyncManager
}

type issuesLoadedMsg []ghapi.Issue

type workflowResultMsg struct {
	message string
	err     error
}

type taskUpdateMsg struct {
	taskID   int
	progress float64
	status   TaskStatus
	err      error
}

type workflowCompleteMsg struct {
	success bool
	message string
}

func initializeModel() model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	// Initialize glamour renderer
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	// Initialize viewport for issue details
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	return model{
		choices:            nil,
		cursor:             0,
		selected:           make(map[int]struct{}),
		spinner:            s,
		totalCount:         0,
		selectedCount:      0,
		viewOffset:         0,
		viewHeight:         15, // Default height, will be adjusted based on screen size
		filterInput:        "",
		filteredChoices:    nil,
		originalIndices:    nil,
		detailViewport:     vp,
		glamourRenderer:    renderer,
		issueContent:       "",
		commandType:        IssuesCommand,
		workflowState:      Loading,
		workflowCursor:     0,
		labelInput:         "",
		projectInput:       "",
		errorMessage:       "",
		tasks:              []WorkflowTask{},
		currentTask:        -1,
		overallProgress:    p,
		githubOpInProgress: false,
		currentActions:     []ghapi.Action{},
	}
}

func initializeModelForIssues(search string) model {
	m := initializeModel()
	m.search = search
	return m
}

func initializeModelForProject(projectID, limit int) model {
	m := initializeModel()
	m.commandType = ProjectCommand
	m.projectID = projectID
	m.limit = limit
	return m
}

func initializeModelForEstimated(projectID, limit int) model {
	m := initializeModel()
	m.commandType = EstimatedCommand
	m.projectID = projectID
	m.limit = limit
	return m
}

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

func (m *model) applyFilter() {
	if m.filterInput == "" {
		// No filter, show all issues
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

func fetchIssues(search string) tea.Cmd {
	return func() tea.Msg {
		issues, err := ghapi.GetIssues(search)
		if err != nil {
			return err
		}
		return issuesLoadedMsg(issues)
	}
}

func fetchProjectItems(projectID, limit int) tea.Cmd {
	return func() tea.Msg {
		items, err := ghapi.GetProjectItems(projectID, limit)
		if err != nil {
			return err
		}
		issues := ghapi.ConvertItemsToIssues(items)
		return issuesLoadedMsg(issues)
	}
}

func fetchEstimatedItems(projectID, limit int) tea.Cmd {
	return func() tea.Msg {
		items, err := ghapi.GetEstimatedTicketsForProject(projectID, limit)
		if err != nil {
			return err
		}
		issues := ghapi.ConvertItemsToIssues(items)
		return issuesLoadedMsg(issues)
	}
}

func (m model) Init() tea.Cmd {
	var fetchCmd tea.Cmd
	switch m.commandType {
	case IssuesCommand:
		fetchCmd = fetchIssues(m.search)
	case ProjectCommand:
		fetchCmd = fetchProjectItems(m.projectID, m.limit)
	case EstimatedCommand:
		fetchCmd = fetchEstimatedItems(m.projectID, m.limit)
	default:
		fetchCmd = fetchIssues("")
	}
	return tea.Batch(fetchCmd, m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

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
				if m.workflowCursor < 4 {
					m.workflowCursor++
				}
			case "k", "up":
				if m.workflowCursor > 0 {
					m.workflowCursor--
				}
			case "enter":
				m.workflowType = WorkflowType(m.workflowCursor)
				switch m.workflowType {
				case BulkAddLabel, BulkRemoveLabel:
					m.workflowState = LabelInput
					m.labelInput = ""
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
	case issuesLoadedMsg:
		m.choices = []ghapi.Issue(msg)
		m.totalCount = len(m.choices)
		m.workflowState = NormalMode
		m.applyFilter()         // Initialize filter state
		m.adjustViewForCursor() // Ensure view is properly initialized
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case progress.FrameMsg:
		progressModel, cmd := m.overallProgress.Update(msg)
		m.overallProgress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	case tea.WindowSizeMsg:
		// Update viewport size when window is resized
		m.detailViewport.Width = msg.Width - 4
		m.detailViewport.Height = msg.Height - 6
	case processTaskMsg:
		// This case is no longer used since we've moved to AsyncManager
		// Ignore these messages
		return m, nil
	case startAsyncWorkflowMsg:
		if m.workflowState == WorkflowRunning {
			// Store the actions and start AsyncManager
			m.currentActions = msg.actions

			// Create status channel
			m.statusChan = make(chan ghapi.Status)

			// Start AsyncManager in a goroutine
			go ghapi.AsyncManager(msg.actions, m.statusChan)

			// Return command to listen for status updates
			return m, m.listenForAsyncStatus()
		}
	case actionStatusMsg:
		if m.workflowState == WorkflowRunning && msg.index < len(m.tasks) {
			// Update the task based on the action status
			if msg.state == "success" {
				m.tasks[msg.index].Status = TaskSuccess
				m.tasks[msg.index].Progress = 1.0
			} else if msg.state == "error" {
				m.tasks[msg.index].Status = TaskError
				m.tasks[msg.index].Progress = 0.0
			}
			m.currentTask = msg.index

			// Update overall progress
			completedTasks := 0
			for _, task := range m.tasks {
				if task.Status == TaskSuccess {
					completedTasks++
				}
			}
			overallPercent := float64(completedTasks) / float64(len(m.tasks))
			cmds = append(cmds, m.overallProgress.SetPercent(overallPercent))

			// Check if we have an error
			if msg.state == "error" {
				m.workflowState = WorkflowComplete
				m.errorMessage = fmt.Sprintf("Task %d failed", msg.index)
			} else if completedTasks == len(m.tasks) {
				// All tasks completed successfully
				m.workflowState = WorkflowComplete
			}
			// Note: Sequential processing is now handled by AsyncManager,
			// so we don't trigger next action processing here
		}
	case asyncStatusMsg:
		if msg.status.State == "done" {
			// AsyncManager is finished, all actions completed
			m.workflowState = WorkflowComplete
			m.statusChan = nil
			return m, nil
		}

		if m.workflowState == WorkflowRunning && msg.status.Index < len(m.tasks) {
			// Update the task based on the async status
			if msg.status.State == "success" {
				m.tasks[msg.status.Index].Status = TaskSuccess
				m.tasks[msg.status.Index].Progress = 1.0
			} else if msg.status.State == "error" {
				m.tasks[msg.status.Index].Status = TaskError
				m.tasks[msg.status.Index].Progress = 0.0
			}
			m.currentTask = msg.status.Index + 1

			// Update overall progress
			completedTasks := 0
			for _, task := range m.tasks {
				if task.Status == TaskSuccess {
					completedTasks++
				}
			}
			overallPercent := float64(completedTasks) / float64(len(m.tasks))
			cmds = append(cmds, m.overallProgress.SetPercent(overallPercent))

			// Check if we have an error
			if msg.status.State == "error" {
				m.workflowState = WorkflowComplete
				m.errorMessage = fmt.Sprintf("Task %d failed", msg.status.Index)
				// Channel will be closed by AsyncManager
				m.statusChan = nil
			} else if completedTasks == len(m.tasks) {
				// All tasks completed successfully
				m.workflowState = WorkflowComplete
				// Channel will be closed by AsyncManager
				m.statusChan = nil
			} else {
				// Continue listening for more status updates
				cmds = append(cmds, m.listenForAsyncStatus())
			}
		}
	case taskUpdateMsg:
		if msg.taskID < len(m.tasks) {
			m.tasks[msg.taskID].Progress = msg.progress
			m.tasks[msg.taskID].Status = msg.status
			if msg.err != nil {
				m.tasks[msg.taskID].Error = msg.err
			}
			m.currentTask = msg.taskID

			// If this is an error, mark workflow as complete
			if msg.status == TaskError {
				m.workflowState = WorkflowComplete
				m.errorMessage = fmt.Sprintf("Task failed: %v", msg.err)
				cmds = append(cmds, m.overallProgress.SetPercent(1.0))
			} else {
				// Update overall progress
				totalProgress := float64(0)
				completedTasks := 0
				for _, task := range m.tasks {
					totalProgress += task.Progress
					if task.Status == TaskSuccess {
						completedTasks++
					}
				}
				overallPercent := totalProgress / float64(len(m.tasks))
				cmds = append(cmds, m.overallProgress.SetPercent(overallPercent))

				// Check if all tasks are completed successfully
				if completedTasks == len(m.tasks) {
					m.workflowState = WorkflowComplete
					// Will show success summary in the view
				} else if msg.status == TaskSuccess && (m.workflowType == BulkAddLabel || m.workflowType == BulkRemoveLabel) {
					// For bulk label operations, trigger the next task with a delay to ensure serialization
					nextTaskID := msg.taskID + 1
					if nextTaskID < len(m.tasks) {
						cmds = append(cmds, func() tea.Msg {
							// Add a small delay before processing the next task
							time.Sleep(50 * time.Millisecond)
							return processTaskMsg{taskID: nextTaskID}
						})
					}
				}
			}
		}
	case workflowCompleteMsg:
		m.workflowState = WorkflowComplete
		if !msg.success {
			m.errorMessage = msg.message
		}
		// Set overall progress to 100%
		cmds = append(cmds, m.overallProgress.SetPercent(1.0))
	case workflowResultMsg:
		// Legacy handler - convert to new system
		if msg.err != nil {
			m.workflowState = WorkflowComplete
			m.errorMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.workflowState = WorkflowComplete
		}
		cmds = append(cmds, m.overallProgress.SetPercent(1.0))
	}

	return m, tea.Batch(cmds...)
}

func truncTitle(title string) string {
	if len(title) > 50 {
		return title[:47] + "..."
	}
	return title
}

// prioritize labels 'story', 'bug', ':release', ':product', '#g-mdm', '#g-orchestration',
// '#g-software'
// show a maximum of 30 characters, if longer, truncate and add '...+n' where n is the number of additional labels
func truncLables(labels []ghapi.Label) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%-35s", "- No Labels -")
	}
	priorityLabels := map[string]bool{"story": true, "bug": true, ":release": true, ":product": true, "#g-mdm": true, "#g-orchestration": true, "#g-software": true}
	var labelNames []string
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}
	displayLabels := []string{}
	secondaryLabels := []string{}
	for _, label := range labelNames {
		if priorityLabels[label] {
			displayLabels = append(displayLabels, label)
		} else {
			secondaryLabels = append(secondaryLabels, label)
		}
	}
	if len(secondaryLabels) > 0 {
		displayLabels = append(displayLabels, secondaryLabels...)
	}
	countDisplay := 0
	sumCharacters := 0
	for _, label := range displayLabels {
		if (sumCharacters + len(label)) > 30 {
			break
		}
		countDisplay += 1
		sumCharacters += len(label) + 2
	}
	extraLabels := 0
	if countDisplay < len(displayLabels) {
		extraLabels = len(displayLabels) - countDisplay
	}
	extraString := ""
	if extraLabels > 0 {
		extraString = fmt.Sprintf("...+%d", extraLabels)
	}
	return fmt.Sprintf("%-35s", fmt.Sprintf("%s%s", strings.Join(displayLabels[:countDisplay], ", "), extraString))
}

func truncEstimate(estimate int) string {
	estString := fmt.Sprintf("%d", estimate)
	if estimate == 0 {
		estString = "-"
	}
	return fmt.Sprintf("%-9s", estString)
}

func truncType(typename string) string {
	if typename == "" {
		typename = "-"
	}
	// Return the typename with spaces filling out to 10 characters
	if len(typename) < 10 {
		return fmt.Sprintf("%-10s", typename)
	}
	return strings.TrimSpace(typename[:10])
}

func (m *model) adjustViewForCursor() {
	currentChoices := m.getCurrentChoices()

	// Ensure cursor is within bounds
	if len(currentChoices) == 0 {
		m.cursor = 0
		m.viewOffset = 0
		return
	}

	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(currentChoices) {
		m.cursor = len(currentChoices) - 1
	}

	// Adjust view offset to keep cursor visible
	if m.cursor < m.viewOffset {
		// Cursor is above visible area, scroll up
		m.viewOffset = m.cursor
	} else if m.cursor >= m.viewOffset+m.viewHeight {
		// Cursor is below visible area, scroll down
		m.viewOffset = m.cursor - m.viewHeight + 1
	}

	// Ensure view offset doesn't go negative or beyond available items
	if m.viewOffset < 0 {
		m.viewOffset = 0
	}
	maxOffset := len(currentChoices) - m.viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}
}

func (m model) View() string {
	s := ""
	switch m.workflowState {
	case Loading:
		var loadingMessage string
		switch m.commandType {
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

	case WorkflowRunning:
		s = "\n🚀 Workflow Execution in Progress\n\n"

		// Show overall progress
		s += fmt.Sprintf("Overall Progress: %s\n\n", m.overallProgress.View())

		// Show individual task progress
		for i, task := range m.tasks {
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

		// Add progress counter at the bottom
		completedTasks := 0
		totalTasks := len(m.tasks)
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

	case WorkflowComplete:
		s = "\n🎉 Workflow Complete!\n\n"

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

	case NormalMode:
		currentChoices := m.getCurrentChoices()
		currentPos := m.cursor + 1
		totalFiltered := len(currentChoices)

		// Header with filter info
		headerText := ""
		if m.filterInput != "" {
			headerText = fmt.Sprintf("GitHub Issues (%d/%d) - Filtered by: '%s':\n\n", currentPos, totalFiltered, m.filterInput)
		} else {
			headerText = fmt.Sprintf("GitHub Issues (%d/%d):\n\n", currentPos, m.totalCount)
		}
		s = headerText + fmt.Sprintf(" %-2d/%-2d Number Estimate  Type       Labels                              Title\n",
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
		s += "Actions: enter/space/x (select), 'o' (view details), '/' (filter), 'w' (workflow), 'q' (quit)\n"
	case IssueDetail:
		if len(m.choices) > 0 && m.cursor < len(m.choices) {
			issue := m.choices[m.cursor]
			s += fmt.Sprintf("Issue #%d Details\n\n", issue.Number)
			s += m.detailViewport.View()
			s += "\n\nNavigation: ↑/↓ or j/k (line), PgUp/PgDn (page), Home/End (top/bottom)\n"
			s += "Actions: 'esc' (back to list), 'q' (quit)\n"
		} else {
			s = "No issue selected\n"
		}
	case FilterInput:
		currentChoices := m.getCurrentChoices()
		totalFiltered := len(currentChoices)

		s = fmt.Sprintf("Filter Issues - %d results:\n", totalFiltered)
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
	case WorkflowSelection:
		s = "\n--- Workflow Selection ---\n"
		workflows := []string{
			"Bulk Add Label",
			"Bulk Remove Label",
			"Bulk Sprint Kickoff",
			"Bulk Milestone Close",
			"Bulk Kick Out Of Sprint",
		}
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
	case LabelInput:
		workflowName := "Add Label"
		if m.workflowType == BulkRemoveLabel {
			workflowName = "Remove Label"
		}
		s = fmt.Sprintf("\n--- %s ---\n", workflowName)
		s += fmt.Sprintf("Label: %s_\n", m.labelInput)
		s += "Press 'enter' to execute, 'esc' to cancel.\n"
	case ProjectInput:
		workflowTitle := "Sprint Kickoff"
		promptText := "Target Project (ID or alias):"
		if m.workflowType == BulkKickOutOfSprint {
			workflowTitle = "Kick Out Of Sprint"
			promptText = "Source Project (ID or alias):"
		}
		s = fmt.Sprintf("\n--- %s ---\n", workflowTitle)
		s += fmt.Sprintf("%s %s_\n", promptText, m.projectInput)
		s += "Press 'enter' to execute, 'esc' to cancel.\n"
	}

	if m.errorMessage != "" && m.workflowState != WorkflowComplete {
		s += fmt.Sprintf("\n%s\n", m.errorMessage)
	}

	return s
}

func (m *model) executeWorkflow() tea.Cmd {
	// Initialize workflow tasks
	m.workflowState = WorkflowRunning
	m.currentTask = 0
	m.tasks = []WorkflowTask{}

	// Collect selected issues
	var selectedIssues []ghapi.Issue
	for i := range m.selected {
		if i < len(m.choices) {
			selectedIssues = append(selectedIssues, m.choices[i])
		}
	}

	if len(selectedIssues) == 0 {
		return func() tea.Msg {
			return workflowCompleteMsg{
				success: false,
				message: "No issues selected",
			}
		}
	}

	// Create actions and tasks based on workflow type
	var actions []ghapi.Action
	switch m.workflowType {
	case BulkAddLabel:
		actions = ghapi.CreateBulkAddLableAction(selectedIssues, m.labelInput)
		// Create individual tasks for each issue
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          i,
				Description: fmt.Sprintf("Adding label '%s' to issue #%d", m.labelInput, issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
	case BulkRemoveLabel:
		actions = ghapi.CreateBulkRemoveLabelAction(selectedIssues, m.labelInput)
		// Create individual tasks for each issue
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          i,
				Description: fmt.Sprintf("Removing label '%s' from issue #%d", m.labelInput, issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
	case BulkSprintKickoff:
		projectID := m.projectID
		if projectID == 0 && m.projectInput != "" {
			// Try to resolve project ID
			resolvedID, err := ghapi.ResolveProjectID(m.projectInput)
			if err != nil {
				return func() tea.Msg {
					return workflowCompleteMsg{
						success: false,
						message: fmt.Sprintf("Failed to resolve project ID: %v", err),
					}
				}
			}
			projectID = resolvedID
		}

		actions = ghapi.CreateBulkSprintKickoffActions(selectedIssues, ghapi.Aliases["draft"], projectID)
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          i,
				Description: fmt.Sprintf("Adding #%d issue to project %d", issue.Number, projectID),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          len(selectedIssues) + i,
				Description: fmt.Sprintf("Adding ':release' label to #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (2 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Syncing estimate fields for #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (3 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Setting current sprint for #%d issue in project %d", issue.Number, projectID),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (4 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Removing ':product' label from #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (5 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Removing #%d issue from drafting project", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
	case BulkMilestoneClose:
		actions = ghapi.CreateBulkMilestoneCloseActions(selectedIssues)
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          len(selectedIssues) + i,
				Description: fmt.Sprintf("Adding #%d issue to drafting project", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          len(selectedIssues) + i,
				Description: fmt.Sprintf("Adding ':product' label to #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          len(selectedIssues) + i,
				Description: fmt.Sprintf("Setting status to 'confirm and celebrate' for #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          len(selectedIssues) + i,
				Description: fmt.Sprintf("Removing ':release' label from #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
	case BulkKickOutOfSprint:
		projectID := m.projectID
		if projectID == 0 && m.projectInput != "" {
			// Try to resolve project ID
			resolvedID, err := ghapi.ResolveProjectID(m.projectInput)
			if err != nil {
				return func() tea.Msg {
					return workflowCompleteMsg{
						success: false,
						message: fmt.Sprintf("Failed to resolve project ID: %v", err),
					}
				}
			}
			projectID = resolvedID
		}

		actions = ghapi.CreateBulkKickOutOfSprintActions(selectedIssues, projectID)
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          i,
				Description: fmt.Sprintf("Adding #%d issue to drafting project", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          len(selectedIssues) + i,
				Description: fmt.Sprintf("Setting status to 'estimated' for #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (2 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Syncing estimate from project %d for #%d issue", projectID, issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (3 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Adding ':product' label to #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (4 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Removing ':release' label from #%d issue", issue.Number),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
		for i, issue := range selectedIssues {
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          (4 * len(selectedIssues)) + i,
				Description: fmt.Sprintf("Removing #%d issue from project %d", issue.Number, projectID),
				Status:      TaskPending,
				Progress:    0.0,
			})
		}
	}

	// For all workflows, start async workflow
	return m.executeAsyncWorkflow(actions)
}

func (m *model) listenForAsyncStatus() tea.Cmd {
	return func() tea.Msg {
		status, ok := <-m.statusChan
		if !ok {
			// Channel is closed, AsyncManager is done
			return asyncStatusMsg{status: ghapi.Status{Index: -1, State: "done"}}
		}
		return asyncStatusMsg{status: status}
	}
}

func (m *model) executeAsyncWorkflow(actions []ghapi.Action) tea.Cmd {
	return func() tea.Msg {
		return startAsyncWorkflowMsg{actions: actions}
	}
}
