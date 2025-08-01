package main

import (
	"fmt"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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

type model struct {
	choices       []ghapi.Issue
	cursor        int
	selected      map[int]struct{}
	spinner       spinner.Model
	totalCount    int
	selectedCount int
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
	tasks           []WorkflowTask
	currentTask     int
	overallProgress progress.Model
	mouseX          int
	mouseY          int
	lastMouseEvent  tea.MouseEvent
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

	return model{
		choices:         nil,
		cursor:          0,
		selected:        make(map[int]struct{}),
		spinner:         s,
		totalCount:      0,
		selectedCount:   0,
		commandType:     IssuesCommand,
		workflowState:   Loading,
		workflowCursor:  0,
		labelInput:      "",
		projectInput:    "",
		errorMessage:    "",
		tasks:           []WorkflowTask{},
		currentTask:     -1,
		overallProgress: p,
		mouseX:          0,
		mouseY:          0,
	}
}

func initializeModelForIssues(search string) model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	return model{
		choices:         nil,
		cursor:          0,
		selected:        make(map[int]struct{}),
		spinner:         s,
		totalCount:      0,
		selectedCount:   0,
		commandType:     IssuesCommand,
		search:          search,
		workflowState:   Loading,
		workflowCursor:  0,
		labelInput:      "",
		projectInput:    "",
		errorMessage:    "",
		tasks:           []WorkflowTask{},
		currentTask:     -1,
		overallProgress: p,
		mouseX:          0,
		mouseY:          0,
	}
}

func initializeModelForProject(projectID, limit int) model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	return model{
		choices:         nil,
		cursor:          0,
		selected:        make(map[int]struct{}),
		spinner:         s,
		totalCount:      0,
		selectedCount:   0,
		commandType:     ProjectCommand,
		projectID:       projectID,
		limit:           limit,
		workflowState:   Loading,
		workflowCursor:  0,
		labelInput:      "",
		projectInput:    "",
		errorMessage:    "",
		tasks:           []WorkflowTask{},
		currentTask:     -1,
		overallProgress: p,
		mouseX:          0,
		mouseY:          0,
	}
}

func initializeModelForEstimated(projectID, limit int) model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	return model{
		choices:         nil,
		cursor:          0,
		selected:        make(map[int]struct{}),
		spinner:         s,
		totalCount:      0,
		selectedCount:   0,
		commandType:     EstimatedCommand,
		projectID:       projectID,
		limit:           limit,
		workflowState:   NormalMode,
		workflowCursor:  0,
		labelInput:      "",
		projectInput:    "",
		errorMessage:    "",
		tasks:           []WorkflowTask{},
		currentTask:     -1,
		overallProgress: p,
		mouseX:          0,
		mouseY:          0,
	}
}

func fetchIssues(search string) tea.Cmd {
	return func() tea.Msg {
		issues, err := ghapi.GetIssues(search)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second) // Simulate a delay for loading
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
		time.Sleep(500 * time.Millisecond) // Brief loading simulation
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
		time.Sleep(500 * time.Millisecond) // Brief loading simulation
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
	case tea.MouseMsg:
		// Store mouse position and event for display
		m.mouseX = msg.X
		m.mouseY = msg.Y
		m.lastMouseEvent = tea.MouseEvent(msg)

		// Handle mouse clicks in workflow running state
		if m.workflowState == WorkflowRunning || m.workflowState == WorkflowComplete {
			// Allow scrolling or clicking on task items
			return m, nil
		}

	case tea.KeyMsg:
		switch m.workflowState {
		case Loading:
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
			case "w":
				if len(m.selected) > 0 {
					m.workflowState = WorkflowSelection
					m.workflowCursor = 0
					m.errorMessage = ""
				}
			case "j", "down":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "k", "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "enter", "x", " ":
				if _, exists := m.selected[m.cursor]; exists {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
				m.selectedCount = len(m.selected)
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case WorkflowSelection:
			switch msg.String() {
			case "j", "down":
				if m.workflowCursor < 3 {
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
				case BulkSprintKickoff:
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
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case progress.FrameMsg:
		progressModel, cmd := m.overallProgress.Update(msg)
		m.overallProgress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
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
		return ""
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

			progressBar := ""
			if task.Status == TaskInProgress || task.Status == TaskSuccess {
				// Create a simple progress bar
				width := 20
				filled := int(task.Progress * float64(width))
				progressBar = "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
			}

			s += fmt.Sprintf("%s %s %s %s", statusIcon, statusText, task.Description, progressBar)

			if task.Error != nil {
				s += fmt.Sprintf(" - Error: %v", task.Error)
			}
			s += "\n"

			// Highlight current task
			if i == m.currentTask {
				s += "   ↑ Current task\n"
			}
		}

		// Show mouse position for demonstration
		if m.mouseX > 0 || m.mouseY > 0 {
			s += fmt.Sprintf("\nMouse: (%d, %d) Event: %s", m.mouseX, m.mouseY, m.lastMouseEvent)
		}
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
				statusText = pendingStyle.Render("UNKNOWN")
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
		s = fmt.Sprintf("GitHub Issues:\n\n %-2d/%-2d Number Estimate  Type       Labels                              Title\n", m.selectedCount, m.totalCount)
		for i, issue := range m.choices {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			selected := ""
			if _, exists := m.selected[i]; exists {
				selected = "[x] "
			} else {
				selected = "[ ] "
			}
			s += fmt.Sprintf("%s %s %s %s %s %s %s\n",
				cursor, selected, fmt.Sprintf("%-6d", issue.Number), truncEstimate(issue.Estimate),
				truncType(issue.Typename), truncLables(issue.Labels), truncTitle(issue.Title))
		}
		s += "\nPress 'j' or 'down' to move down, 'k' or 'up' to move up, 'enter' to select/deselect, 'w' to run workflow, and 'q' to quit.\n"
	case WorkflowSelection:
		s = "\n--- Workflow Selection ---\n"
		workflows := []string{
			"Bulk Add Label",
			"Bulk Remove Label",
			"Bulk Sprint Kickoff",
			"Bulk Milestone Close",
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
		s = "\n--- Sprint Kickoff ---\n"
		s += fmt.Sprintf("Target Project (ID or alias): %s_\n", m.projectInput)
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

	// Define tasks based on workflow type
	switch m.workflowType {
	case BulkAddLabel:
		m.tasks = []WorkflowTask{
			{ID: 0, Description: fmt.Sprintf("Adding label '%s' to %d issues", m.labelInput, len(selectedIssues)), Status: TaskPending, Progress: 0.0},
		}
	case BulkRemoveLabel:
		m.tasks = []WorkflowTask{
			{ID: 0, Description: fmt.Sprintf("Removing label '%s' from %d issues", m.labelInput, len(selectedIssues)), Status: TaskPending, Progress: 0.0},
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

		m.tasks = []WorkflowTask{
			{ID: 0, Description: fmt.Sprintf("Adding %d issues to project %d", len(selectedIssues), projectID), Status: TaskPending, Progress: 0.0},
			{ID: 1, Description: "Adding ':release' label to issues", Status: TaskPending, Progress: 0.0},
			{ID: 2, Description: "Syncing estimate fields", Status: TaskPending, Progress: 0.0},
			{ID: 3, Description: "Setting current sprint", Status: TaskPending, Progress: 0.0},
			{ID: 4, Description: "Removing ':product' label from issues", Status: TaskPending, Progress: 0.0},
			{ID: 5, Description: "Removing issues from drafting project", Status: TaskPending, Progress: 0.0},
		}
	case BulkMilestoneClose:
		m.tasks = []WorkflowTask{
			{ID: 0, Description: fmt.Sprintf("Adding %d issues to drafting project", len(selectedIssues)), Status: TaskPending, Progress: 0.0},
			{ID: 1, Description: "Adding ':product' label to issues", Status: TaskPending, Progress: 0.0},
			{ID: 2, Description: "Setting status to 'confirm and celebrate'", Status: TaskPending, Progress: 0.0},
			{ID: 3, Description: "Removing ':release' label from issues", Status: TaskPending, Progress: 0.0},
		}
	}

	return m.runWorkflowTasks(selectedIssues)
}

func (m *model) runWorkflowTasks(selectedIssues []ghapi.Issue) tea.Cmd {
	return func() tea.Msg {
		projectID := m.projectID
		if projectID == 0 && m.projectInput != "" {
			resolvedID, err := ghapi.ResolveProjectID(m.projectInput)
			if err != nil {
				return workflowCompleteMsg{
					success: false,
					message: fmt.Sprintf("Failed to resolve project ID: %v", err),
				}
			}
			projectID = resolvedID
		}

		// Execute workflow with progress updates
		switch m.workflowType {
		case BulkAddLabel:
			return executeBulkAddLabel(selectedIssues, m.labelInput)
		case BulkRemoveLabel:
			return executeBulkRemoveLabel(selectedIssues, m.labelInput)
		case BulkSprintKickoff:
			return executeBulkSprintKickoff(selectedIssues, projectID)
		case BulkMilestoneClose:
			return executeBulkMilestoneClose(selectedIssues)
		}

		return workflowCompleteMsg{
			success: false,
			message: "Unknown workflow type",
		}
	}
}

func executeBulkAddLabel(selectedIssues []ghapi.Issue, label string) tea.Msg {
	// Start task
	time.Sleep(100 * time.Millisecond) // Simulate some work

	err := ghapi.BulkAddLabel(selectedIssues, label)
	if err != nil {
		return taskUpdateMsg{taskID: 0, progress: 0.0, status: TaskError, err: err}
	}

	// Mark task as complete first
	return taskUpdateMsg{taskID: 0, progress: 1.0, status: TaskSuccess, err: nil}
}

func executeBulkRemoveLabel(selectedIssues []ghapi.Issue, label string) tea.Msg {
	time.Sleep(100 * time.Millisecond) // Simulate some work

	err := ghapi.BulkRemoveLabel(selectedIssues, label)
	if err != nil {
		return taskUpdateMsg{taskID: 0, progress: 0.0, status: TaskError, err: err}
	}

	// Mark task as complete first
	return taskUpdateMsg{taskID: 0, progress: 1.0, status: TaskSuccess, err: nil}
}

func executeBulkSprintKickoff(selectedIssues []ghapi.Issue, projectID int) tea.Msg {
	// This would be better implemented as a series of commands, but for now keep it simple
	err := ghapi.BulkSprintKickoff(selectedIssues, projectID)
	if err != nil {
		return workflowCompleteMsg{
			success: false,
			message: fmt.Sprintf("Sprint kickoff failed: %v", err),
		}
	}

	return workflowCompleteMsg{
		success: true,
		message: fmt.Sprintf("Executed sprint kickoff for %d issues to project %d", len(selectedIssues), projectID),
	}
}

func executeBulkMilestoneClose(selectedIssues []ghapi.Issue) tea.Msg {
	err := ghapi.BulkMilestoneClose(selectedIssues)
	if err != nil {
		return workflowCompleteMsg{
			success: false,
			message: fmt.Sprintf("Milestone close failed: %v", err),
		}
	}

	return workflowCompleteMsg{
		success: true,
		message: fmt.Sprintf("Executed milestone close for %d issues", len(selectedIssues)),
	}
}
