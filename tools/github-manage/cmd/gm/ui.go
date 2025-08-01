package main

import (
	"fmt"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type CommandType int

const (
	IssuesCommand CommandType = iota
	ProjectCommand
	EstimatedCommand
)

type model struct {
	choices       []ghapi.Issue
	cursor        int
	selected      map[int]struct{}
	loading       bool
	spinner       spinner.Model
	totalCount    int
	selectedCount int
	// Command-specific parameters
	commandType CommandType
	projectID   int
	limit       int
}

type issuesLoadedMsg []ghapi.Issue

func initializeModel() model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())
	return model{
		choices:       nil,
		cursor:        0,
		selected:      make(map[int]struct{}),
		loading:       true,
		spinner:       s,
		totalCount:    0,
		selectedCount: 0,
		commandType:   IssuesCommand,
	}
}

func initializeModelForProject(projectID, limit int) model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())
	return model{
		choices:       nil,
		cursor:        0,
		selected:      make(map[int]struct{}),
		loading:       true,
		spinner:       s,
		totalCount:    0,
		selectedCount: 0,
		commandType:   ProjectCommand,
		projectID:     projectID,
		limit:         limit,
	}
}

func initializeModelForEstimated(projectID, limit int) model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())
	return model{
		choices:       nil,
		cursor:        0,
		selected:      make(map[int]struct{}),
		loading:       true,
		spinner:       s,
		totalCount:    0,
		selectedCount: 0,
		commandType:   EstimatedCommand,
		projectID:     projectID,
		limit:         limit,
	}
}

func fetchIssues() tea.Cmd {
	return func() tea.Msg {
		issues, err := ghapi.GetIssues("")
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
		fetchCmd = fetchIssues()
	case ProjectCommand:
		fetchCmd = fetchProjectItems(m.projectID, m.limit)
	case EstimatedCommand:
		fetchCmd = fetchEstimatedItems(m.projectID, m.limit)
	default:
		fetchCmd = fetchIssues()
	}
	return tea.Batch(fetchCmd, m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process key messages while loading
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "a":
			// add label

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
	case issuesLoadedMsg:
		m.choices = []ghapi.Issue(msg)
		m.totalCount = len(m.choices)
		m.loading = false
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
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
	if m.loading {
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
	}

	s := fmt.Sprintf("GitHub Issues:\n\n %-2d/%-2d Number Estimate  Type       Labels                              Title\n", m.selectedCount, m.totalCount)
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
	s += "\nPress 'j' or 'down' to move down, 'k' or 'up' to move up, 'enter' to select/deselect, and 'q' to quit.\n"
	return s
}
