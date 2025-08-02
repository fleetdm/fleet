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

	return model{
		choices:            nil,
		cursor:             0,
		selected:           make(map[int]struct{}),
		spinner:            s,
		totalCount:         0,
		selectedCount:      0,
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

func fetchIssues(search string) tea.Cmd {
	return func() tea.Msg {
		// Ensure GitHub API calls don't overlap
		time.Sleep(100 * time.Millisecond)

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
		// Ensure GitHub API calls don't overlap
		time.Sleep(100 * time.Millisecond)

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
		// Ensure GitHub API calls don't overlap
		time.Sleep(100 * time.Millisecond)

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
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case progress.FrameMsg:
		progressModel, cmd := m.overallProgress.Update(msg)
		m.overallProgress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
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

	// For BulkAddLabel and BulkRemoveLabel, start async workflow
	return m.executeAsyncWorkflow(actions)
}

func (m *model) processAction(actions []ghapi.Action, index int) tea.Cmd {
	return func() tea.Msg {
		if index >= len(actions) {
			// All actions completed
			return workflowCompleteMsg{
				success: true,
				message: fmt.Sprintf("Successfully processed %d actions", len(actions)),
			}
		}

		action := actions[index]

		// Add delay to ensure no concurrent GitHub operations
		time.Sleep(100 * time.Millisecond)

		// Process the action based on its type
		var err error
		switch action.Type {
		case ghapi.ATAddLabel:
			err = ghapi.AddLabelToIssue(action.Issue.Number, action.Label)
		case ghapi.ATRemoveLabel:
			err = ghapi.RemoveLabelFromIssue(action.Issue.Number, action.Label)
		case ghapi.ATAddIssueToProject:
			err = ghapi.AddIssueToProject(action.Issue.Number, action.Project)
		case ghapi.ATRemoveIssueFromProject:
			err = ghapi.RemoveIssueFromProject(action.Issue.Number, action.Project)
		case ghapi.ATSetStatus:
			err = ghapi.SetIssueStatus(action.Issue.Number, action.Project, action.Status)
		case ghapi.ATSyncEstimate:
			err = ghapi.SyncEstimateField(action.Issue.Number, action.SourceProject, action.Project)
		case ghapi.ATSetSprint:
			err = ghapi.SetCurrentSprint(action.Issue.Number, action.Project)
		default:
			err = fmt.Errorf("unknown action type: %s", action.Type)
		}

		if err != nil {
			return actionStatusMsg{
				index: index,
				state: "error",
			}
		}

		return actionStatusMsg{
			index: index,
			state: "success",
		}
	}
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

		// Execute workflow with progress updates using AsyncManager
		switch m.workflowType {
		case BulkAddLabel:
			return m.executeAsyncWorkflow(ghapi.CreateBulkAddLableAction(selectedIssues, m.labelInput))
		case BulkRemoveLabel:
			return m.executeAsyncWorkflow(ghapi.CreateBulkRemoveLabelAction(selectedIssues, m.labelInput))
		case BulkSprintKickoff:
			return executeBulkSprintKickoff(selectedIssues, projectID)
		case BulkMilestoneClose:
			return executeBulkMilestoneClose(selectedIssues)
		case BulkKickOutOfSprint:
			return executeBulkKickOutOfSprint(selectedIssues, projectID)
		}

		return workflowCompleteMsg{
			success: false,
			message: "Unknown workflow type",
		}
	}
}

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

func executeBulkSprintKickoff(selectedIssues []ghapi.Issue, projectID int) tea.Msg {
	// Add a delay to ensure no overlap with other GitHub commands
	time.Sleep(200 * time.Millisecond)

	// Get the drafting project ID for source
	draftingProjectID := 0 // You may need to resolve this from aliases

	// This would be better implemented as a series of commands, but for now keep it simple
	err := ghapi.BulkSprintKickoff(selectedIssues, draftingProjectID, projectID)
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
	// Add a delay to ensure no overlap with other GitHub commands
	time.Sleep(200 * time.Millisecond)

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

func executeBulkKickOutOfSprint(selectedIssues []ghapi.Issue, sourceProjectID int) tea.Msg {
	// Add a delay to ensure no overlap with other GitHub commands
	time.Sleep(200 * time.Millisecond)

	err := ghapi.BulkKickOutOfSprint(selectedIssues, sourceProjectID)
	if err != nil {
		return workflowCompleteMsg{
			success: false,
			message: fmt.Sprintf("Kick out of sprint failed: %v", err),
		}
	}

	return workflowCompleteMsg{
		success: true,
		message: fmt.Sprintf("Executed kick out of sprint for %d issues from project %d", len(selectedIssues), sourceProjectID),
	}
}
