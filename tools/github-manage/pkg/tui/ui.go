package tui

import (
	"fmt"
	"sort"
	"strings"

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
	SprintCommand
	MilestoneCommand
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

// Adding new workflow needs type, values
// And support in switch cases
// Search code for newworkflow
type WorkflowType int

const (
	BulkAddLabel WorkflowType = iota
	BulkRemoveLabel
	BulkSprintKickoff
	BulkMilestoneClose
	BulkKickOutOfSprint
	BulkDemoSummary
	BulkMoveToCurrentSprint
)

var WorkflowTypeValues = []string{
	"Bulk Add Label",
	"Bulk Remove Label",
	"Bulk Sprint Kickoff",
	"Bulk Milestone Close",
	"Bulk Kick Out Of Sprint",
	"Bulk Demo Summary",
	"Move to current sprint",
}

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
	choices         []ghapi.Issue
	cursor          int
	selected        map[int]struct{}
	relatedCache    map[int][]int // issue number -> related issue numbers
	spinner         spinner.Model
	totalCount      int
	totalAvailable  int // reported total items in project (may exceed totalCount if limited)
	rawFetchedCount int // number of items actually fetched before mode-specific filtering
	selectedCount   int
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
	// Terminal sizing
	termWidth int
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
	exitMessage        string
	// Headless run hints: select every issue and/or run a workflow automatically
	// once the view is populated (see RunTUI).
	autoSelectAll bool
	autoWorkflow  string
}

// workflowAliases maps a --workflow flag value to a WorkflowType for headless
// runs. Only workflows that can run without further prompting are listed (label
// workflows need interactive text input, so they're intentionally excluded).
var workflowAliases = map[string]WorkflowType{
	"demo":            BulkDemoSummary,
	"demo-summary":    BulkDemoSummary,
	"milestone-close": BulkMilestoneClose,
	"sprint-kickoff":  BulkSprintKickoff,
	"kick-out":        BulkKickOutOfSprint,
	"move-to-sprint":  BulkMoveToCurrentSprint,
}

// resolveWorkflow maps a --workflow flag value to its WorkflowType, matching
// case-insensitively. The bool is false when the name isn't a known workflow.
func resolveWorkflow(name string) (WorkflowType, bool) {
	wt, ok := workflowAliases[strings.ToLower(strings.TrimSpace(name))]
	return wt, ok
}

type issuesLoadedMsg struct {
	issues         []ghapi.Issue
	totalAvailable int
	rawFetched     int
}

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

type workflowExitMsg struct {
	success bool
	message string
}

// RunTUI launches the interactive TUI. autoSelectAll selects every issue as soon
// as the view is populated, and autoWorkflow (a name like "demo") runs that
// workflow immediately instead of waiting for the user to pick one — together
// they enable headless runs such as `gm project apple -aw demo`.
func RunTUI(commandType CommandType, projectID int, limit int, search string, autoSelectAll bool, autoWorkflow string) {
	var mm model
	switch commandType {
	case ProjectCommand:
		mm = initializeModelForProject(projectID, limit)
	case EstimatedCommand:
		mm = initializeModelForEstimated(projectID, limit)
	case SprintCommand:
		mm = initializeModelForSprint(projectID, limit)
	case IssuesCommand:
		mm = initializeModelForIssues(search)
	case MilestoneCommand:
		mm = initializeModelForMilestone(search, limit)
	default:
		// error and exit
		fmt.Println("Unsupported command type for TUI")
		return
	}
	// Carry over the search string as a generic mode hint (e.g., for sprint: "previous")
	mm.search = search
	mm.autoSelectAll = autoSelectAll
	mm.autoWorkflow = autoWorkflow
	p := tea.NewProgram(&mm)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running Bubble Tea program: %v\n", err)
	}
	if mm.exitMessage != "" {
		fmt.Println(mm.exitMessage)
	}
}

// base model w/ global defaults
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
		relatedCache:       make(map[int][]int),
		spinner:            s,
		totalCount:         0,
		totalAvailable:     0,
		rawFetchedCount:    0,
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

// Each view should have it's own initialize model to
// add details to help the queries newview
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

func initializeModelForSprint(projectID, limit int) model {
	m := initializeModel()
	m.commandType = SprintCommand
	m.projectID = projectID
	m.limit = limit
	return m
}

func initializeModelForMilestone(title string, limit int) model {
	m := initializeModel()
	m.commandType = MilestoneCommand
	m.limit = limit
	m.search = title // reuse search to carry milestone title
	return m
}

// newview Add a corresponding issue fetcher
func fetchIssues(search string) tea.Cmd {
	return func() tea.Msg {
		issues, err := ghapi.GetIssues(search)
		if err != nil {
			return err
		}
		return issuesLoadedMsg{issues: issues, totalAvailable: len(issues), rawFetched: len(issues)}
	}
}

func fetchProjectItems(projectID, limit int) tea.Cmd {
	return func() tea.Msg {
		items, total, err := ghapi.GetProjectItemsWithTotal(projectID, limit)
		if err != nil {
			return err
		}
		issues := ghapi.ConvertItemsToIssues(items)
		return issuesLoadedMsg{issues: issues, totalAvailable: total, rawFetched: limit}
	}
}

func fetchEstimatedItems(projectID, limit int) tea.Cmd {
	return func() tea.Msg {
		items, total, err := ghapi.GetEstimatedTicketsForProjectWithTotal(projectID, limit)
		if err != nil {
			return err
		}
		issues := ghapi.ConvertItemsToIssues(items)
		// totalAvailable reflects total items in drafting project; rawFetched is the fetch limit
		return issuesLoadedMsg{issues: issues, totalAvailable: total, rawFetched: limit}
	}
}

func fetchSprintItems(projectID, limit int) tea.Cmd {
	return func() tea.Msg {
		items, total, err := ghapi.GetCurrentSprintItemsWithTotal(projectID, limit)
		if err != nil {
			return err
		}
		issues := ghapi.ConvertItemsToIssues(items)
		return issuesLoadedMsg{issues: issues, totalAvailable: total, rawFetched: limit}
	}
}

func fetchPreviousSprintItems(projectID, limit int) tea.Cmd {
	return func() tea.Msg {
		items, total, err := ghapi.GetPreviousSprintItemsWithTotal(projectID, limit)
		if err != nil {
			return err
		}
		issues := ghapi.ConvertItemsToIssues(items)
		return issuesLoadedMsg{issues: issues, totalAvailable: total, rawFetched: limit}
	}
}

func fetchMilestoneItems(title string, limit int) tea.Cmd {
	return func() tea.Msg {
		// Get open-issue count for this milestone to drive warning banner
		totalCount, cntErr := ghapi.GetMilestoneOpenIssueCount(title)
		// Fetch up to 'limit' open issues for the milestone
		issues, exceeded, err := ghapi.GetIssuesByMilestoneLimited(title, limit)
		if err != nil {
			return err
		}
		// Prefer accurate total from GraphQL; fallback to limit+1 sentinel when cntErr occurs
		total := totalCount
		if cntErr != nil {
			total = len(issues)
			if exceeded {
				total = limit + 1
			}
		}
		return issuesLoadedMsg{issues: issues, totalAvailable: total, rawFetched: limit}
	}
}

// newview add command type / fetcher to switch
func (m model) Init() tea.Cmd {
	var fetchCmd tea.Cmd
	switch m.commandType {
	case IssuesCommand:
		fetchCmd = fetchIssues(m.search)
	case ProjectCommand:
		fetchCmd = fetchProjectItems(m.projectID, m.limit)
	case EstimatedCommand:
		fetchCmd = fetchEstimatedItems(m.projectID, m.limit)
	case SprintCommand:
		// Use the search field as a mode hint; when "previous", fetch previous sprint instead of current
		if strings.EqualFold(strings.TrimSpace(m.search), "previous") || strings.EqualFold(strings.TrimSpace(m.search), "prev") {
			fetchCmd = fetchPreviousSprintItems(m.projectID, m.limit)
		} else {
			fetchCmd = fetchSprintItems(m.projectID, m.limit)
		}
	case MilestoneCommand:
		fetchCmd = fetchMilestoneItems(m.search, m.limit)
	default:
		fetchCmd = fetchIssues("")
	}
	return tea.Batch(fetchCmd, m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.HandleHotkeys(msg)
	default:
		return m.HandleStateChange(msg)
	}
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
		s = m.RenderLoading()

	case WorkflowRunning:
		s = m.RenderWorkflowRunning()

	case WorkflowComplete:
		s = m.RenderWorkflowComplete()

	case NormalMode:
		s = m.RenderIssueTable()

	case IssueDetail:
		s = m.RenderIssueDetail()

	case FilterInput:
		s = m.RenderFilterInput()

	case WorkflowSelection:
		s = m.RenderWorkflowSelection()

	case LabelInput:
		s = m.RenderLabelInput()

	case ProjectInput:
		s = m.RenderProjectInput()
	}

	if m.errorMessage != "" && m.workflowState != WorkflowComplete {
		s += fmt.Sprintf("\nError: %s\n", m.errorMessage)
	}

	return s
}

// writeDemoSummarySection writes a "Features Completed" / "Bugs Completed"
// breakdown for the given issues to builder, grouping each category by assignee
// (falling back to the assignee's login, then "unassigned"). Issues are
// classified by their labels: "story" → feature, "bug" → bug, with an
// "~unreleased bug" label demoting a bug back out of the summary.
func writeDemoSummarySection(builder *strings.Builder, issues []ghapi.Issue) {
	assigneeOrUnassigned := func(issue ghapi.Issue) string {
		if len(issue.Assignees) > 0 {
			user, err := ghapi.GetUserName(issue.Assignees[0].Login)
			if err == nil {
				if user.Name != "" {
					return user.Name
				}
				return user.Login
			}
		}
		return "unassigned"
	}

	features := map[string][]ghapi.Issue{}
	bugs := map[string][]ghapi.Issue{}
	for _, issue := range issues {
		isFeature := false
		isBug := false
		for _, l := range issue.Labels {
			if l.Name == "story" {
				isFeature = true
			}
			if l.Name == "bug" {
				isBug = true
			}
			if l.Name == "~unreleased bug" {
				isBug = false
			}
		}
		assignee := assigneeOrUnassigned(issue)
		if isFeature {
			features[assignee] = append(features[assignee], issue)
		}
		if isBug {
			bugs[assignee] = append(bugs[assignee], issue)
		}
	}

	writeCategory := func(heading string, byAssignee map[string][]ghapi.Issue) {
		builder.WriteString(heading)
		if len(byAssignee) == 0 {
			builder.WriteString("_None_\n\n")
		}
		var assignees []string
		for a := range byAssignee {
			assignees = append(assignees, a)
		}
		sort.Strings(assignees)
		for _, a := range assignees {
			builder.WriteString(fmt.Sprintf("%s\n", a))
			for _, issue := range byAssignee[a] {
				builder.WriteString(fmt.Sprintf("[#%d](https://github.com/fleetdm/fleet/issues/%d)%s\n", issue.Number, issue.Number, issue.Title))
			}
			builder.WriteString("\n")
		}
	}

	writeCategory("## Features Completed\n\n", features)
	writeCategory("## Bugs Completed\n\n", bugs)
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
	// newworkflow add workflow steps here (actions / tasks)
	case BulkDemoSummary:
		// Build summary markdown grouped by milestone, then (within each milestone)
		// by label category and assignee. Issues with no milestone fall under a
		// trailing "No milestone" section.
		const noMilestone = "No milestone"
		byMilestone := map[string][]ghapi.Issue{}
		var milestoneOrder []string
		seenMilestone := map[string]bool{}
		// isSkippedStatus reports whether an issue in this status is not demo-ready
		// (still in the inbox/drafting/blocked/ready columns) and should be excluded
		// from the summary. Matched loosely so emoji/prefix drift ("🥚 Ready") still
		// hits.
		isSkippedStatus := func(status string) bool {
			s := strings.ToLower(strings.TrimSpace(status))
			if strings.HasSuffix(s, "ready") {
				return true
			}
			for _, kw := range []string{"inbox", "drafting", "blocked"} {
				if strings.Contains(s, kw) {
					return true
				}
			}
			return false
		}
		var skipped []ghapi.Issue // selected but excluded due to their status
		for _, issue := range selectedIssues {
			if issue.Status == "" {
				continue
			}
			if isSkippedStatus(issue.Status) {
				skipped = append(skipped, issue)
				continue
			}
			ms := noMilestone
			if issue.Milestone != nil && issue.Milestone.Title != "" {
				ms = issue.Milestone.Title
			}
			if !seenMilestone[ms] {
				seenMilestone[ms] = true
				milestoneOrder = append(milestoneOrder, ms)
			}
			byMilestone[ms] = append(byMilestone[ms], issue)
		}
		// Sort milestones by title, keeping "No milestone" last.
		sort.Slice(milestoneOrder, func(i, j int) bool {
			a, b := milestoneOrder[i], milestoneOrder[j]
			if a == noMilestone {
				return false
			}
			if b == noMilestone {
				return true
			}
			return a < b
		})

		var builder strings.Builder
		for _, ms := range milestoneOrder {
			builder.WriteString(fmt.Sprintf("# %s\n\n", ms))
			writeDemoSummarySection(&builder, byMilestone[ms])
		}
		// List the selected issues we excluded and why, so the skip is visible
		// rather than silent. Sorted by status, then issue number.
		if len(skipped) > 0 {
			sort.Slice(skipped, func(i, j int) bool {
				if skipped[i].Status != skipped[j].Status {
					return skipped[i].Status < skipped[j].Status
				}
				return skipped[i].Number < skipped[j].Number
			})
			builder.WriteString("# Skipped (not demo-ready)\n\n")
			for _, issue := range skipped {
				builder.WriteString(fmt.Sprintf("[#%d](https://github.com/fleetdm/fleet/issues/%d) %s — %s\n", issue.Number, issue.Number, issue.Title, issue.Status))
			}
			builder.WriteString("\n")
		}
		return func() tea.Msg { return workflowExitMsg{success: true, message: builder.String()} }
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
	case BulkMoveToCurrentSprint:
		projectID := m.projectID
		if projectID == 0 && m.projectInput != "" {
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
		if projectID == 0 {
			return func() tea.Msg {
				return workflowCompleteMsg{
					success: false,
					message: "Project ID is required for this workflow",
				}
			}
		}
		actions = ghapi.CreateBulkMoveToCurrentSprintIfNotReadyQA(selectedIssues, projectID)
		for i, issue := range selectedIssues {
			// Only tasks for those that will be acted on; description indicates conditional nature
			desc := fmt.Sprintf("If not 'ready' or 'qa', set current sprint for #%d in project %d", issue.Number, projectID)
			m.tasks = append(m.tasks, WorkflowTask{
				ID:          i,
				Description: desc,
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
