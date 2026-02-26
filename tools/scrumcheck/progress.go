package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	clrReset  = "\033[0m"
	clrDim    = "\033[2m"
	clrCyan   = "\033[36m"
	clrBlue   = "\033[34m"
	clrGreen  = "\033[32m"
	clrYellow = "\033[33m"
	clrRed    = "\033[31m"
	clrGray   = "\033[90m"
)

type phaseStatus int

const (
	phasePending phaseStatus = iota
	phaseRunning
	phaseDone
	phaseWarn
	phaseFail
)

type phaseEntry struct {
	name    string
	status  phaseStatus
	summary string
}

type phaseTracker struct {
	mu         sync.Mutex
	phases     []phaseEntry
	globalRow  int
	phaseRow   int
	footerRow  int
	logRow     int
	statusText string
	logLines   []string

	bridgeOpsStarted int
	bridgeOpsDone    int
	bridgeOpsOK      int
	bridgeOpsErr     int
	bridgeOpsTotal   time.Duration
}

// newPhaseTracker initializes progress state and paints the initial console UI.
func newPhaseTracker(names []string) *phaseTracker {
	phases := make([]phaseEntry, 0, len(names))
	for _, name := range names {
		phases = append(phases, phaseEntry{name: name, status: phasePending})
	}
	p := &phaseTracker{
		phases:    phases,
		globalRow: 4,
		phaseRow:  6,
		footerRow: 6 + len(phases) + 1,
		logRow:    6 + len(phases) + 3,
		logLines:  make([]string, 0, 24),
	}
	p.clearScreen()
	p.renderAll()
	return p
}

// clearScreen resets the terminal before drawing the tracker UI.
func (p *phaseTracker) clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// renderAll prints the full tracker layout (header, phases, footer, and log).
func (p *phaseTracker) renderAll() {
	fmt.Printf("%s🚀 scrumcheck flight console%s\n", clrCyan, clrReset)
	fmt.Printf("%sMission control online. Tracking scan phases in real time.%s\n\n", clrDim, clrReset)
	fmt.Println(p.renderGlobalLine())
	fmt.Println()
	for i := range p.phases {
		fmt.Println(p.renderPhaseLine(i))
	}
	fmt.Println()
	fmt.Println(p.renderFooterLine())
	fmt.Printf("\033[%d;1H%sBridge activity log%s\n", p.logRow, clrDim, clrReset)
	fmt.Printf("\033[%d;1H", p.footerRow)
}

// renderPhaseLine renders a single phase row with icon, bar, and summary.
func (p *phaseTracker) renderPhaseLine(i int) string {
	entry := p.phases[i]
	icon, color, bar := p.phaseVisual(entry.status)
	text := entry.name
	if entry.summary != "" {
		text = text + " " + clrDim + "| " + entry.summary + clrReset
	}
	return fmt.Sprintf("%s[%d/%d]%s %s%s%s %s", clrBlue, i+1, len(p.phases), clrReset, color, icon, clrReset, bar+" "+text)
}

// renderGlobalLine renders the global progress bar and completed phase count.
func (p *phaseTracker) renderGlobalLine() string {
	completed := p.completedCount()
	total := len(p.phases)
	ratio := float64(completed) / float64(max(total, 1))
	color := stageColor(ratio)
	bar := coloredBar(42, completed, total, color)
	return fmt.Sprintf("%sMission progress%s %s %s%d/%d%s",
		clrBlue, clrReset,
		bar,
		clrDim, completed, total, clrReset,
	)
}

// renderFooterLine renders status text plus bridge operation summary.
func (p *phaseTracker) renderFooterLine() string {
	opsLine := p.renderBridgeOpsLine()
	if p.statusText == "" {
		return fmt.Sprintf("%sStanding by...%s  %s", clrDim, clrReset, opsLine)
	}
	return p.statusText + "  " + opsLine
}

// redrawPhase refreshes one phase row and dependent aggregate/footer rows.
func (p *phaseTracker) redrawPhase(i int) {
	row := p.phaseRow + i
	fmt.Printf("\033[%d;1H\033[2K%s", row, p.renderPhaseLine(i))
	p.redrawGlobal()
	p.redrawFooter()
}

// redrawGlobal refreshes only the global progress row in-place.
func (p *phaseTracker) redrawGlobal() {
	fmt.Printf("\033[%d;1H\033[2K%s", p.globalRow, p.renderGlobalLine())
}

// redrawFooter refreshes only the footer/status row in-place.
func (p *phaseTracker) redrawFooter() {
	fmt.Printf("\033[%d;1H\033[2K%s", p.footerRow, p.renderFooterLine())
	fmt.Printf("\033[%d;1H", p.footerRow)
}

// phaseStart marks a phase as running and updates the UI.
func (p *phaseTracker) phaseStart(i int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseRunning
	p.phases[i].summary = ""
	p.redrawPhase(i)
}

// phaseDone marks a phase successful and sets its summary text.
func (p *phaseTracker) phaseDone(i int, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseDone
	p.phases[i].summary = summary
	p.redrawPhase(i)
}

// phaseWarn marks a phase as warning and sets its summary text.
func (p *phaseTracker) phaseWarn(i int, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseWarn
	p.phases[i].summary = summary
	p.redrawPhase(i)
}

// phaseFail marks a phase as failed and sets its summary text.
func (p *phaseTracker) phaseFail(i int, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseFail
	p.phases[i].summary = summary
	p.redrawPhase(i)
}

// waitingForBrowser updates footer status while waiting for browser bridge use.
func (p *phaseTracker) waitingForBrowser(reportPath string) {
	_ = reportPath
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusText = fmt.Sprintf("%s🛰️  Waiting for report bridge...%s",
		clrCyan,
		clrReset,
	)
	p.redrawFooter()
}

// showReportLink is intentionally a no-op for the current tracker UI.
func (p *phaseTracker) showReportLink(reportURL string) {
	_ = reportURL
}

// bridgeListening updates UI when the bridge server starts listening.
func (p *phaseTracker) bridgeListening(baseURL string, idleTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusText = fmt.Sprintf(
		"%s🛸 UI uplink listening on %s%s%s | idle timeout %s%s%s | Press Ctrl+C to close",
		clrCyan, clrBlue, baseURL, clrReset, clrYellow, idleTimeout.Round(time.Minute).String(), clrReset,
	)
	p.appendBridgeLogLocked("🛸 listening at " + baseURL)
	p.redrawFooter()
}

// bridgeSignal records bridge operation events and updates counters/log/footer.
func (p *phaseTracker) bridgeSignal(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if evt, ok := parseBridgeOpSignal(msg); ok {
		if evt.Stage == "start" {
			p.bridgeOpsStarted++
		}
		if evt.Stage == "done" {
			p.bridgeOpsDone++
			if evt.Status == "ok" {
				p.bridgeOpsOK++
			}
			if evt.Status == "error" {
				p.bridgeOpsErr++
			}
			if evt.Elapsed > 0 {
				p.bridgeOpsTotal += evt.Elapsed
			}
		}
		p.statusText = fmt.Sprintf("%s🔧 %s%s", clrCyan, evt.summary(), clrReset)
		p.appendBridgeLogLocked("🔧 " + evt.summary())
	} else {
		p.statusText = fmt.Sprintf("%s%s%s", clrCyan, msg, clrReset)
		p.appendBridgeLogLocked(msg)
	}
	p.redrawFooter()
}

// bridgeStopped marks bridge shutdown state and appends a shutdown log entry.
func (p *phaseTracker) bridgeStopped(reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusText = fmt.Sprintf("%s🧯 UI uplink offline: %s%s", clrDim, reason, clrReset)
	p.appendBridgeLogLocked("🧯 stopped: " + reason)
	p.redrawFooter()
	fmt.Println()
}

// appendBridgeLogLocked appends a timestamped bridge log line to the UI.
func (p *phaseTracker) appendBridgeLogLocked(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	p.logLines = append(p.logLines, line)
	row := p.logRow + len(p.logLines)
	fmt.Printf("\033[%d;1H\033[2K%s%s%s", row, clrDim, line, clrReset)
	fmt.Printf("\033[%d;1H", p.footerRow)
}

// renderBridgeOpsLine summarizes bridge operation throughput and latency.
func (p *phaseTracker) renderBridgeOpsLine() string {
	if p.bridgeOpsStarted == 0 {
		return fmt.Sprintf("%sbridge ops idle%s", clrDim, clrReset)
	}
	ratio := float64(p.bridgeOpsDone) / float64(max(p.bridgeOpsStarted, 1))
	bar := coloredBar(20, p.bridgeOpsDone, p.bridgeOpsStarted, stageColor(ratio))
	avg := "-"
	if p.bridgeOpsDone > 0 && p.bridgeOpsTotal > 0 {
		avg = shortDuration(time.Duration(int64(p.bridgeOpsTotal) / int64(p.bridgeOpsDone)))
	}
	return fmt.Sprintf(
		"%sbridge ops%s %s %s%d/%d%s %sok=%d err=%d avg=%s%s",
		clrBlue,
		clrReset,
		bar,
		clrDim, p.bridgeOpsDone, p.bridgeOpsStarted, clrReset,
		clrDim, p.bridgeOpsOK, p.bridgeOpsErr, avg, clrReset,
	)
}

type bridgeOpEvent struct {
	Caller  string
	Op      string
	Stage   string
	Status  string
	Repo    string
	Issue   string
	Elapsed time.Duration
}

// summary formats a human-readable bridge operation event summary.
func (e bridgeOpEvent) summary() string {
	target := e.Repo
	if strings.TrimSpace(e.Issue) != "" && e.Issue != "0" {
		target += "#" + e.Issue
	}
	elapsed := "-"
	if e.Elapsed > 0 {
		elapsed = shortDuration(e.Elapsed)
	}
	return fmt.Sprintf(
		"%s %s %s (%s, caller=%s, %s)",
		e.Op,
		e.Stage,
		target,
		e.Status,
		e.Caller,
		elapsed,
	)
}

// parseBridgeOpSignal parses structured BRIDGE_OP messages from bridge handlers.
func parseBridgeOpSignal(msg string) (bridgeOpEvent, bool) {
	raw := strings.TrimSpace(msg)
	if !strings.HasPrefix(raw, "BRIDGE_OP ") {
		return bridgeOpEvent{}, false
	}
	evt := bridgeOpEvent{}
	fields := strings.Fields(strings.TrimPrefix(raw, "BRIDGE_OP "))
	for _, f := range fields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		switch key {
		case "caller":
			evt.Caller = val
		case "op":
			evt.Op = val
		case "stage":
			evt.Stage = val
		case "status":
			evt.Status = val
		case "repo":
			evt.Repo = val
		case "issue":
			evt.Issue = val
		case "item":
			evt.Issue = val
		case "elapsed":
			if d, err := time.ParseDuration(val); err == nil {
				evt.Elapsed = d
			}
		}
	}
	if evt.Op == "" || evt.Stage == "" {
		return bridgeOpEvent{}, false
	}
	if evt.Status == "" {
		evt.Status = "working"
	}
	if evt.Repo == "" {
		evt.Repo = "item"
	}
	if evt.Caller == "" {
		evt.Caller = "unknown"
	}
	return evt, true
}

// phaseVisual maps phase state to icon/color/progress-bar styling.
func (p *phaseTracker) phaseVisual(status phaseStatus) (icon, color, bar string) {
	ratio := float64(p.completedCount()) / float64(max(len(p.phases), 1))
	runColor := stageColor(ratio)
	switch status {
	case phaseRunning:
		return "🟠", clrYellow, runningPhaseBar(30, runColor)
	case phaseDone:
		return "🟢", clrGreen, donePhaseBar(30)
	case phaseWarn:
		return "🟠", clrYellow, donePhaseBar(30)
	case phaseFail:
		return "🔴", clrRed, donePhaseBar(30)
	default:
		return "⚪", clrDim, pendingPhaseBar(30)
	}
}

// completedCount returns the number of phases in terminal (non-running) states.
func (p *phaseTracker) completedCount() int {
	count := 0
	for _, ph := range p.phases {
		if ph.status == phaseDone || ph.status == phaseWarn || ph.status == phaseFail {
			count++
		}
	}
	return count
}

// shortDuration rounds durations for compact status and log output.
func shortDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(10 * time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}

// countAwaitingViolations totals Awaiting QA violations across projects.
func countAwaitingViolations(m map[int][]Item) int {
	total := 0
	for _, items := range m {
		total += len(items)
	}
	return total
}

// countStaleViolations totals stale Awaiting QA violations across projects.
func countStaleViolations(m map[int][]StaleAwaitingViolation) int {
	total := 0
	for _, items := range m {
		total += len(items)
	}
	return total
}

// phaseSummaryKV joins key/value summary segments with a shared delimiter.
func phaseSummaryKV(pairs ...string) string {
	return strings.Join(pairs, " | ")
}

// pendingPhaseBar renders an empty phase progress bar.
func pendingPhaseBar(width int) string {
	return "[" + clrGray + strings.Repeat("░", width) + clrReset + "]"
}

// donePhaseBar renders a fully completed phase progress bar.
func donePhaseBar(width int) string {
	return "[" + clrGreen + strings.Repeat("█", width) + clrReset + "]"
}

// runningPhaseBar renders a partial progress bar for active phases.
func runningPhaseBar(width int, fillColor string) string {
	fill := max(1, width/3)
	return "[" + fillColor + strings.Repeat("█", fill) + clrGray + strings.Repeat("░", width-fill) + clrReset + "]"
}

// coloredBar renders a proportional progress bar from done/total counts.
func coloredBar(width, done, total int, fillColor string) string {
	if total <= 0 {
		return pendingPhaseBar(width)
	}
	fill := int(float64(done) / float64(total) * float64(width))
	if done > 0 && fill == 0 {
		fill = 1
	}
	if fill > width {
		fill = width
	}
	return "[" + fillColor + strings.Repeat("█", fill) + clrGray + strings.Repeat("░", width-fill) + clrReset + "]"
}

// stageColor chooses red/yellow/green based on completion ratio.
func stageColor(ratio float64) string {
	switch {
	case ratio < 0.34:
		return clrRed
	case ratio < 0.67:
		return clrYellow
	default:
		return clrGreen
	}
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
