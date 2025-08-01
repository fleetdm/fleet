package main

import (
	"fmt"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type model struct {
	choices       []ghapi.Issue
	cursor        int
	selected      map[int]struct{}
	loading       bool
	spinner       spinner.Model
	totalCount    int
	selectedCount int
}

type issuesLoadedMsg []ghapi.Issue

func initializeModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())
	return model{
		choices:       nil,
		cursor:        0,
		selected:      make(map[int]struct{}),
		loading:       true,
		spinner:       s,
		totalCount:    0,
		selectedCount: 0,
	}
}

func initializeModelWithIssues(issues []ghapi.Issue) model {
	s := spinner.New()
	s.Spinner = spinner.Monkey
	s.Style = s.Style.Foreground(spinner.New().Style.GetForeground())
	return model{
		choices:  issues,
		cursor:   0,
		selected: make(map[int]struct{}),
		loading:  false,
		spinner:  s,
	}
}

func fetchIssues() tea.Cmd {
	return func() tea.Msg {
		issues, err := ghapi.GetIssues("")
		if err != nil {
			return err
		}
		time.Sleep(3 * time.Second) // Simulate a delay for loading
		return issuesLoadedMsg(issues)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchIssues(), m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process key messages while loading
		if m.loading {
			return m, nil
		}
		switch msg.String() {
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
		return fmt.Sprintf("\n%s Fetching Issues...\n\n", m.spinner.View())
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

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Get GitHub issues",
	Run: func(cmd *cobra.Command, args []string) {
		model := initializeModel()
		p := tea.NewProgram(&model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		fmt.Printf("Selected issues: ")
		for i := range model.selected {
			if i < len(model.choices) {
				fmt.Printf("#%d ", model.choices[i].Number)
			}
		}
		// fmt.Printf("\n")
		// fmt.Printf("debug issues: %+v\n", model.choices)
		// fmt.Printf("debug selected: %+v\n", model.selected)
	},
}
