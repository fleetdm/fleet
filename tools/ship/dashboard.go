package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// dashboardModel is the steady-state TUI: header, status block, log panes,
// keybind hint row. For PR 1 the data is mostly placeholder; later PRs wire
// it to real fleet/webpack output and runtime state.
type dashboardModel struct {
	worktree   string
	branch     string
	database   string
	port       int
	publicURL  string
	uptime     string
	fleetLogs  []string // ring buffer
	webpackLog string
}

func newDashboardModel() dashboardModel {
	return dashboardModel{
		worktree:  "—",
		branch:    "—",
		database:  "—",
		port:      8080,
		publicURL: "(not yet started)",
		uptime:    "—",
	}
}

func (d dashboardModel) view(width int, state runState) string {
	if width <= 0 {
		width = 80
	}

	header := renderHeader(width, state, d.uptime)
	status := renderStatusBlock(d)
	fleetPane := renderLogPane(width, "fleet server", d.fleetLogs, 5)
	webpackPane := renderLogPane(width, "webpack", strings.Split(d.webpackLog, "\n"), 2)
	hints := renderHints()

	body := strings.Join([]string{
		header,
		"",
		status,
		"",
		fleetPane,
		"",
		webpackPane,
		"",
		hints,
	}, "\n")

	return stylePane.Width(width - 2).Render(body)
}

func renderHeader(width int, state runState, uptime string) string {
	left := styleHeaderBrand.Render("Fleet ship")
	right := stateStyle(state).Render(state.String())
	if uptime != "" && uptime != "—" {
		right = right + styleHint.Render("  ·  uptime "+uptime)
	}
	pad := width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func renderStatusBlock(d dashboardModel) string {
	row := func(label, value string) string {
		return styleLabel.Render(fmt.Sprintf("%-10s", label)) + styleValue.Render(value)
	}
	urlRow := styleLabel.Render(fmt.Sprintf("%-10s", "public")) + styleURL.Render(d.publicURL)
	return strings.Join([]string{
		row("worktree", d.worktree) + "    " + row("branch", d.branch),
		row("database", d.database) + "    " + row("port", fmt.Sprintf("%d", d.port)),
		urlRow,
	}, "\n")
}

func renderLogPane(width int, title string, lines []string, maxLines int) string {
	header := styleSection.Render("─ " + title + " " + strings.Repeat("─", max(0, width-len(title)-6)))
	if len(lines) == 0 {
		lines = []string{styleHint.Render("(waiting for output...)")}
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return header + "\n" + strings.Join(lines, "\n")
}

func renderHints() string {
	hints := []struct {
		key, desc string
	}{
		{"r", "rebuild"},
		{"p", "pause"},
		{"s", "switch"},
		{"t", "hosts"},
		{"l", "logs"},
		{"d", "debug"},
		{"?", "help"},
		{"q", "quit"},
	}
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, styleKey.Render(h.key)+" "+styleHint.Render(h.desc))
	}
	return strings.Join(parts, styleHint.Render(" · "))
}
