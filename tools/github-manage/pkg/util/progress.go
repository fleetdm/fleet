package util

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
)

// ProgressBar renders an inline progress bar to stderr for long per-item loops
// (e.g. fetching metadata for each issue in a milestone report). It writes only
// to stderr, so it never corrupts report output sent to stdout or a file. When
// stderr is not a terminal the bar stays silent to avoid polluting piped or
// redirected output.
type ProgressBar struct {
	label   string
	total   int
	width   int
	start   time.Time
	enabled bool
}

// NewProgressBar creates a progress bar for total steps with the given label.
func NewProgressBar(label string, total int) *ProgressBar {
	fd := os.Stderr.Fd()
	return &ProgressBar{
		label:   label,
		total:   total,
		width:   30,
		start:   time.Now(),
		enabled: isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd),
	}
}

// Update redraws the bar at the given 1-based step. detail is shown after the
// counter, e.g. the issue number currently being fetched ("#46029").
func (p *ProgressBar) Update(current int, detail string) {
	if p == nil || !p.enabled || p.total <= 0 {
		return
	}
	if current < 0 {
		current = 0
	}
	if current > p.total {
		current = p.total
	}
	frac := float64(current) / float64(p.total)
	filled := int(frac * float64(p.width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	line := fmt.Sprintf("%s [%s] %d/%d (%.0f%%)", p.label, bar, current, p.total, frac*100)
	if detail != "" {
		line += " " + detail
	}
	if eta := p.eta(current); eta != "" {
		line += " " + eta
	}

	// \r returns to column 0; \033[K clears leftovers from a longer prior line.
	fmt.Fprintf(os.Stderr, "\r\033[K%s", line)
}

// eta estimates remaining time from the average duration per completed step.
func (p *ProgressBar) eta(current int) string {
	if current <= 0 || current >= p.total {
		return ""
	}
	per := time.Since(p.start) / time.Duration(current)
	remaining := (per * time.Duration(p.total-current)).Round(time.Second)
	return fmt.Sprintf("ETA %s", remaining)
}

// Done terminates the bar's line so subsequent output starts cleanly. It is a
// no-op when the bar is disabled.
func (p *ProgressBar) Done() {
	if p == nil || !p.enabled {
		return
	}
	fmt.Fprintln(os.Stderr)
}
