package main

import (
	"fmt"

	"fleetdm/gm/pkg/ghapi"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type model struct {
	choices  []ghapi.Issue
	cursor   int
	selected map[int]struct{}
}

func initializeModel(issues []ghapi.Issue) model {
	return model{
		choices:  issues,
		cursor:   0,
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if _, exists := m.selected[m.cursor]; exists {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func truncTitle(title string) string {
	if len(title) > 50 {
		return title[:47] + "..."
	}
	return title
}

func truncLables(labels []ghapi.Label) string {
	if len(labels) == 0 {
		return ""
	}
	var labelNames []string
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}
	if len(labelNames) > 3 {
		return fmt.Sprintf("%s, %s, %s...", labelNames[0], labelNames[1], labelNames[2])
	}
	return fmt.Sprintf("%s", labelNames)
}

func (m model) View() string {
	s := "GitHub Issues:\n\n"
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
		s += fmt.Sprintf("%s %s %d %s %s\n", cursor, selected, issue.Number, truncLables(issue.Labels), truncTitle(issue.Title))
	}
	s += "\nPress 'j' or 'down' to move down, 'k' or 'up' to move up, 'enter' to select/deselect, and 'q' to quit.\n"
	return s
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Get GitHub issues",
	Run: func(cmd *cobra.Command, args []string) {
		issues, err := ghapi.GetIssues()
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			return
		}
		model := initializeModel(issues)
		p := tea.NewProgram(model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		fmt.Printf("Selected issues: ")
		for i := range model.selected {
			if i < len(model.choices) {
				fmt.Printf("#%d ", model.choices[i].Number)
			}
		}
	},
}
