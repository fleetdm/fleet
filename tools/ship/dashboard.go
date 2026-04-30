package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// stepRow is one entry in the dashboard's start-sequence step list.
type stepRow struct {
	Kind    stepKind
	Status  stepStatus
	Detail  string
	Elapsed time.Duration
}

const (
	maxFleetLogLines   = 8
	maxWebpackLogLines = 3
	maxNgrokLogLines   = 2
)

// dashboardModel is the steady-state TUI: header, status block, log panes,
// keybind hint row. Until the engine reports stateRunning, the log panes
// area is taken over by the start-sequence step list.
type dashboardModel struct {
	worktree  string
	branch    string
	database  string
	port      int
	publicURL string
	startedAt time.Time

	steps    []stepRow
	stepByID map[stepKind]int // index into steps

	fleetLog   []string
	webpackLog []string
	ngrokLog   []string
	startLog   []string // process output during the build steps
}

func newDashboardModel() dashboardModel {
	return dashboardModel{
		worktree:  "—",
		branch:    "—",
		database:  "—",
		port:      8080,
		publicURL: "(starting...)",
		stepByID:  map[stepKind]int{},
	}
}

// beginStart is called when the engine is launched. We pre-populate the step
// list with all expected steps in pending state so the user sees the full
// plan from the moment the dashboard appears.
func (d *dashboardModel) beginStart() {
	expected := []stepKind{
		stepDockerUp, stepMakeDeps, stepMakeBuild, stepGenerateDev,
		stepPrepareDB, stepServe, stepNgrok,
	}
	d.steps = make([]stepRow, 0, len(expected))
	d.stepByID = make(map[stepKind]int, len(expected))
	for i, k := range expected {
		d.steps = append(d.steps, stepRow{Kind: k, Status: stepPending})
		d.stepByID[k] = i
	}
	d.startLog = nil
}

func (d *dashboardModel) applyStepUpdate(msg stepUpdateMsg) {
	idx, ok := d.stepByID[msg.Kind]
	if !ok {
		return
	}
	d.steps[idx].Status = msg.Status
	d.steps[idx].Detail = msg.Detail
	d.steps[idx].Elapsed = msg.Elapsed
}

func (d *dashboardModel) appendLog(line logLine) {
	switch line.Source {
	case "fleet":
		d.fleetLog = pushRing(d.fleetLog, line.Line, maxFleetLogLines)
	case "webpack":
		d.webpackLog = pushRing(d.webpackLog, line.Line, maxWebpackLogLines)
	case "ngrok":
		d.ngrokLog = pushRing(d.ngrokLog, line.Line, maxNgrokLogLines)
	default:
		// "start" — output from one-shot commands during bring-up.
		d.startLog = pushRing(d.startLog, line.Line, 5)
	}
}

func (d *dashboardModel) markRunning(ngrokDomain string, startedAt time.Time) {
	d.startedAt = startedAt
	if ngrokDomain != "" {
		d.publicURL = "https://" + ngrokDomain
	}
}

func pushRing(buf []string, line string, max int) []string {
	buf = append(buf, line)
	if len(buf) > max {
		buf = buf[len(buf)-max:]
	}
	return buf
}

// view renders the dashboard. errMsg, when non-empty, is shown above the
// keybind row so the user notices a fatal problem.
func (d dashboardModel) view(width int, state runState, errMsg string) string {
	if width <= 0 {
		width = 80
	}

	header := renderHeader(width, state, d.uptimeStr())

	body := []string{
		header,
		"",
		renderStatusBlock(d),
		"",
	}

	if state == stateBuilding || state == stateError {
		body = append(body, renderStepList(d))
	} else {
		body = append(body, renderLogPane(width, "fleet server", d.fleetLog, maxFleetLogLines))
		body = append(body, "")
		body = append(body, renderLogPane(width, "webpack", d.webpackLog, maxWebpackLogLines))
	}

	if errMsg != "" {
		body = append(body, "", lipgloss.NewStyle().Foreground(colorErr).Render("✗ "+errMsg))
	}

	body = append(body, "", renderHints())
	return stylePane.Width(width - 2).Render(strings.Join(body, "\n"))
}

func (d dashboardModel) uptimeStr() string {
	if d.startedAt.IsZero() {
		return ""
	}
	return formatUptime(time.Since(d.startedAt))
}

func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

func renderHeader(width int, state runState, uptime string) string {
	left := styleHeaderBrand.Render("Fleet ship")
	right := stateStyle(state).Render(state.String())
	if uptime != "" {
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

func renderStepList(d dashboardModel) string {
	rows := make([]string, 0, len(d.steps))
	for _, s := range d.steps {
		rows = append(rows, renderStepRow(s))
	}
	if len(d.startLog) > 0 {
		rows = append(rows, "")
		rows = append(rows, styleHint.Render("recent output:"))
		for _, line := range d.startLog {
			rows = append(rows, styleHint.Render("  "+line))
		}
	}
	return strings.Join(rows, "\n")
}

func renderStepRow(s stepRow) string {
	icon, name := stepIconAndName(s)
	right := ""
	switch s.Status {
	case stepDone:
		right = styleHint.Render("  " + formatElapsed(s.Elapsed))
	case stepFailed:
		right = lipgloss.NewStyle().Foreground(colorErr).Render("  " + s.Detail)
	case stepRunning:
		right = styleHint.Render("  ...")
	}
	return "  " + icon + "  " + name + right
}

func stepIconAndName(s stepRow) (string, string) {
	switch s.Status {
	case stepDone:
		return lipgloss.NewStyle().Foreground(colorOK).Bold(true).Render("✓"), s.Kind.String()
	case stepRunning:
		return lipgloss.NewStyle().Foreground(colorWarn).Bold(true).Render("◐"), s.Kind.String()
	case stepFailed:
		return lipgloss.NewStyle().Foreground(colorErr).Bold(true).Render("✗"), s.Kind.String()
	case stepSkipped:
		return styleHint.Render("·"), styleHint.Render(s.Kind.String())
	default:
		return styleHint.Render("·"), styleHint.Render(s.Kind.String())
	}
}

func formatElapsed(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func renderLogPane(width int, title string, lines []string, _ int) string {
	header := styleSection.Render("─ " + title + " " + strings.Repeat("─", maxOrZero(width-len(title)-6)))
	if len(lines) == 0 {
		lines = []string{styleHint.Render("(waiting for output...)")}
	}
	return header + "\n" + strings.Join(lines, "\n")
}

func maxOrZero(n int) int {
	if n < 0 {
		return 0
	}
	return n
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
