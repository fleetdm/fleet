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

// We keep a much larger ring buffer per source than the dashboard itself
// will ever render — the on-demand log overlay screen reads from these,
// so the cap needs to be useful, not just glanceable.
const logBufferLines = 200

// dashboardModel is the steady-state TUI: header, status block, services
// list, keybind hint row. Until the engine reports stateRunning, the
// services area is taken over by the start-sequence step list. Logs are
// captured per-source and shown via the log overlay screen, not on the
// dashboard itself.
type dashboardModel struct {
	worktree  string
	branch    string
	database  string
	port      int
	publicURL string
	startedAt time.Time

	steps    []stepRow
	stepByID map[stepKind]int

	// trigger is shown as a "trigger:" row above the step list during a
	// rebuild. Empty during the initial bring-up.
	trigger string

	// queued is the count of file changes accumulated while paused.
	// Surfaced in the keybind/status row so the user can see why their
	// edits aren't applying yet.
	queued int

	fleetLog   []string
	webpackLog []string
	ngrokLog   []string
	startLog   []string
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

// setIdentity is called once at startup with values that don't change for
// the life of the run.
func (d *dashboardModel) setIdentity(worktree, branch, database string) {
	if worktree != "" {
		d.worktree = worktree
	}
	if branch != "" {
		d.branch = branch
	}
	if database != "" {
		d.database = database
	}
}

// beginStart pre-populates the step list with all expected steps in
// pending state so the dashboard renders the full plan from the moment
// it appears.
func (d *dashboardModel) beginStart() {
	d.populateSteps([]stepKind{
		stepDockerUp, stepMakeDeps, stepGenerateDev, stepMakeBuild,
		stepPrepareDB, stepServe, stepNgrok,
	})
	d.startLog = nil
	d.trigger = ""
}

// beginRebuild pre-populates the step list with the rebuild sequence's
// steps and stores the trigger reason for the dashboard to render above
// the list.
func (d *dashboardModel) beginRebuild(reason string) {
	d.populateSteps([]stepKind{
		stepStopFleet, stepMakeBuild, stepPrepareDB, stepServe,
	})
	d.trigger = reason
	d.startLog = nil
}

func (d *dashboardModel) populateSteps(expected []stepKind) {
	d.steps = make([]stepRow, 0, len(expected))
	d.stepByID = make(map[stepKind]int, len(expected))
	for i, k := range expected {
		d.steps = append(d.steps, stepRow{Kind: k, Status: stepPending})
		d.stepByID[k] = i
	}
}

// setQueued records the count of paused-while-queued files for the
// keybind/status row.
func (d *dashboardModel) setQueued(n int) { d.queued = n }

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
		d.fleetLog = pushRing(d.fleetLog, line.Line, logBufferLines)
	case "webpack":
		d.webpackLog = pushRing(d.webpackLog, line.Line, logBufferLines)
	case "ngrok":
		d.ngrokLog = pushRing(d.ngrokLog, line.Line, logBufferLines)
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

	body := []string{
		renderHeader(width, state, d.uptimeStr()),
		"",
		renderStatusBlock(d),
		"",
	}

	if state == stateBuilding || state == stateError {
		body = append(body, renderStepList(d))
	} else {
		body = append(body, renderServices())
	}

	if errMsg != "" {
		body = append(body, "", lipgloss.NewStyle().Foreground(colorErr).Render("✗ "+errMsg))
	}

	body = append(body, "", renderHints(state, d.queued))
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
		row("worktree", d.worktree),
		row("branch", d.branch),
		row("database", d.database),
		urlRow,
	}, "\n")
}

// renderServices is the compact view shown once Fleet is up. Each row is
// a status dot + service name + short description. Detailed output lives
// behind l/w/n keybinds.
func renderServices() string {
	dot := lipgloss.NewStyle().Foreground(colorOK).Bold(true).Render("●")
	row := func(name, status string) string {
		return "    " + dot + "  " +
			lipgloss.NewStyle().Width(16).Render(name) +
			styleHint.Render(status)
	}
	return strings.Join([]string{
		styleLabel.Render("Services"),
		row("Fleet server", "running"),
		row("Webpack", "watching for frontend changes"),
		row("ngrok", "tunnel up"),
	}, "\n")
}

func renderStepList(d dashboardModel) string {
	rows := make([]string, 0, len(d.steps)+2)
	if d.trigger != "" {
		rows = append(rows, styleHint.Render("trigger: ")+styleValue.Render(d.trigger))
		rows = append(rows, "")
	}
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

// renderHints shows only the keybinds that are wired and contextually
// relevant. r/p/l/w/n appear once Fleet is up; r is hidden while paused
// because manual rebuilds during a demo would defeat the pause.
func renderHints(state runState, queued int) string {
	type hint struct{ key, desc string }
	var hints []hint
	switch state {
	case stateRunning:
		hints = append(hints,
			hint{"r", "rebuild"},
			hint{"p", "pause"},
			hint{"l", "fleet logs"},
			hint{"w", "webpack logs"},
			hint{"n", "ngrok traffic"},
		)
	case statePaused:
		hints = append(hints,
			hint{"p", "resume"},
			hint{"l", "fleet logs"},
			hint{"w", "webpack logs"},
			hint{"n", "ngrok traffic"},
		)
	}
	hints = append(hints, hint{"q", "quit"})

	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, styleKey.Render(h.key)+" "+styleHint.Render(h.desc))
	}
	row := strings.Join(parts, styleHint.Render(" · "))

	if state == statePaused && queued > 0 {
		row += "    " + styleHint.Render(fmt.Sprintf("%d file%s queued", queued, plural(queued)))
	}
	return row
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
