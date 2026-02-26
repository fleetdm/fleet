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
}

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

func (p *phaseTracker) clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func (p *phaseTracker) renderAll() {
	fmt.Printf("%s🚀 qacheck flight console%s\n", clrCyan, clrReset)
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

func (p *phaseTracker) renderPhaseLine(i int) string {
	entry := p.phases[i]
	icon, color, bar := p.phaseVisual(entry.status)
	text := entry.name
	if entry.summary != "" {
		text = text + " " + clrDim + "| " + entry.summary + clrReset
	}
	return fmt.Sprintf("%s[%d/%d]%s %s%s%s %s", clrBlue, i+1, len(p.phases), clrReset, color, icon, clrReset, bar+" "+text)
}

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

func (p *phaseTracker) renderFooterLine() string {
	if p.statusText == "" {
		return fmt.Sprintf("%sStanding by...%s", clrDim, clrReset)
	}
	return p.statusText
}

func (p *phaseTracker) redrawPhase(i int) {
	row := p.phaseRow + i
	fmt.Printf("\033[%d;1H\033[2K%s", row, p.renderPhaseLine(i))
	p.redrawGlobal()
	p.redrawFooter()
}

func (p *phaseTracker) redrawGlobal() {
	fmt.Printf("\033[%d;1H\033[2K%s", p.globalRow, p.renderGlobalLine())
}

func (p *phaseTracker) redrawFooter() {
	fmt.Printf("\033[%d;1H\033[2K%s", p.footerRow, p.renderFooterLine())
	fmt.Printf("\033[%d;1H", p.footerRow)
}

func (p *phaseTracker) phaseStart(i int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseRunning
	p.phases[i].summary = ""
	p.redrawPhase(i)
}

func (p *phaseTracker) phaseDone(i int, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseDone
	p.phases[i].summary = summary
	p.redrawPhase(i)
}

func (p *phaseTracker) phaseWarn(i int, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseWarn
	p.phases[i].summary = summary
	p.redrawPhase(i)
}

func (p *phaseTracker) phaseFail(i int, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phases[i].status = phaseFail
	p.phases[i].summary = summary
	p.redrawPhase(i)
}

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

func (p *phaseTracker) showReportLink(reportURL string) {
	_ = reportURL
}

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

func (p *phaseTracker) bridgeSignal(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusText = fmt.Sprintf("%s%s%s", clrCyan, msg, clrReset)
	p.appendBridgeLogLocked(msg)
	p.redrawFooter()
}

func (p *phaseTracker) bridgeStopped(reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusText = fmt.Sprintf("%s🧯 UI uplink offline: %s%s", clrDim, reason, clrReset)
	p.appendBridgeLogLocked("🧯 stopped: " + reason)
	p.redrawFooter()
	fmt.Println()
}

func (p *phaseTracker) appendBridgeLogLocked(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	p.logLines = append(p.logLines, line)
	row := p.logRow + len(p.logLines)
	fmt.Printf("\033[%d;1H\033[2K%s%s%s", row, clrDim, line, clrReset)
	fmt.Printf("\033[%d;1H", p.footerRow)
}

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

func (p *phaseTracker) completedCount() int {
	count := 0
	for _, ph := range p.phases {
		if ph.status == phaseDone || ph.status == phaseWarn || ph.status == phaseFail {
			count++
		}
	}
	return count
}

func shortDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(10 * time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}

func countAwaitingViolations(m map[int][]Item) int {
	total := 0
	for _, items := range m {
		total += len(items)
	}
	return total
}

func countStaleViolations(m map[int][]StaleAwaitingViolation) int {
	total := 0
	for _, items := range m {
		total += len(items)
	}
	return total
}

func phaseSummaryKV(pairs ...string) string {
	return strings.Join(pairs, " | ")
}

func pendingPhaseBar(width int) string {
	return "[" + clrGray + strings.Repeat("░", width) + clrReset + "]"
}

func donePhaseBar(width int) string {
	return "[" + clrGreen + strings.Repeat("█", width) + clrReset + "]"
}

func runningPhaseBar(width int, fillColor string) string {
	fill := max(1, width/3)
	return "[" + fillColor + strings.Repeat("█", fill) + clrGray + strings.Repeat("░", width-fill) + clrReset + "]"
}

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
