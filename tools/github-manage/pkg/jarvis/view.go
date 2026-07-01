package jarvis

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleBarStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#444444"))
	reasonStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	prTagStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF"))
	issueTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A371F7"))
	sessTagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E3B341"))
	noticeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
)

func (m Model) View() string {
	switch m.state {
	case stateLoading:
		return fmt.Sprintf("\n  %s Summoning your work from GitHub…\n", m.spinner.View())
	case stateError:
		return "\n" + errStyle.Render("  Jarvis hit an error.") + "\n" +
			fmt.Sprintf("  %v\n\n", m.err) +
			dimStyle.Render("  Check `gh auth status` and your network, then press r to retry · q to quit") + "\n"
	}
	return m.renderBoard()
}

func (m Model) renderBoard() string {
	var b strings.Builder

	title := fmt.Sprintf("🎩 Jarvis · @%s · %d shown · refreshed %s",
		m.login, m.filtered.Total(), m.lastRefresh.Format("15:04:05"))
	b.WriteString(titleBarStyle.Render(title))
	b.WriteString("\n")

	if m.notice != "" {
		style := noticeStyle
		if m.noticeErr {
			style = errStyle
		}
		b.WriteString(style.Render("  " + m.notice))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.filtered.Total() == 0 {
		b.WriteString(reasonStyle.Render("  Inbox zero — nothing is waiting on you. 🎉"))
		b.WriteString("\n\n")
		b.WriteString(m.footer())
		return b.String()
	}

	now := time.Now()
	var lines []string
	cursorLine := 0
	itemIdx := 0
	for _, bk := range BucketOrder {
		items := m.filtered.Buckets[bk]
		if len(items) == 0 {
			continue
		}
		lines = append(lines, m.bucketHeader(bk, len(items)))
		for _, it := range items {
			if itemIdx == m.cursor {
				cursorLine = len(lines)
			}
			hiddenLabel := ""
			if !m.triage.Visible(m.key(it), it.Updated, now) {
				hiddenLabel = m.triage.Label(m.key(it))
			}
			lines = append(lines, m.itemLine(it, itemIdx == m.cursor, hiddenLabel))
			itemIdx++
		}
		lines = append(lines, "")
	}

	viewport := m.height - 6
	if viewport < 3 {
		viewport = 3
	}
	start := 0
	if len(lines) > viewport {
		if cursorLine >= viewport {
			start = cursorLine - viewport + 1
		}
		if start+viewport > len(lines) {
			start = len(lines) - viewport
		}
		if start < 0 {
			start = 0
		}
	}
	end := start + viewport
	if end > len(lines) {
		end = len(lines)
	}
	b.WriteString(strings.Join(lines[start:end], "\n"))
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m Model) bucketHeader(bk Bucket, n int) string {
	return headerStyle.Render(fmt.Sprintf("%s (%d)", bk.Title(), n)) + " " + subtitleStyle.Render(bk.Subtitle())
}

func (m Model) itemLine(it Item, selected bool, hiddenLabel string) string {
	kind := "PR"
	switch it.Kind {
	case KindIssue:
		kind = "issue"
	case KindSession:
		kind = "claude"
	}
	num := ""
	if it.Number > 0 {
		num = fmt.Sprintf("#%d", it.Number)
	}
	age := humanizeAge(it.Updated)
	title := truncate(it.Title, m.titleWidth())
	marker := ""
	if it.HasSession {
		marker = " 💬"
	}
	reason := it.Reason
	if hiddenLabel != "" {
		reason = hiddenLabel
	}

	if selected {
		plain := fmt.Sprintf("▸ %-6s %-7s %s%s   %s   %s", kind, num, title, marker, reason, age)
		return selectedStyle.Render(plain)
	}

	label := fmt.Sprintf("%-6s", kind)
	var styledLabel string
	switch it.Kind {
	case KindIssue:
		styledLabel = issueTagStyle.Render(label)
	case KindSession:
		styledLabel = sessTagStyle.Render(label)
	default:
		styledLabel = prTagStyle.Render(label)
	}
	reasonStyled := reasonStyle.Render(reason)
	if hiddenLabel != "" {
		reasonStyled = dimStyle.Render(reason)
	}
	return fmt.Sprintf("  %s %-7s %s%s   %s   %s",
		styledLabel, num, title, marker, reasonStyled, dimStyle.Render(age))
}

func (m Model) footer() string {
	switch m.mode {
	case modeSnooze:
		return dimStyle.Render("snooze: ") +
			"[1] 1 hour  [2] 4 hours  [3] tomorrow  [4] 1 week  " +
			dimStyle.Render("· esc cancel")
	case modeConfirmMerge:
		n := ""
		if it, ok := m.currentItem(); ok {
			n = fmt.Sprintf("#%d", it.Number)
		}
		return errStyle.Render(fmt.Sprintf("Merge %s with squash? ", n)) +
			"[y] yes  " + dimStyle.Render("· any other key cancels")
	case modeComment:
		return m.commentInput.View() + "\n" + dimStyle.Render("enter post · esc cancel")
	default:
		hidden := ""
		if m.hidden > 0 {
			state := "show"
			if m.showHidden {
				state = "hide"
			}
			hidden = dimStyle.Render(fmt.Sprintf(" · %d hidden (H to %s)", m.hidden, state))
		}
		help := "↑/↓ move · enter open · m merge · c comment · s snooze · d dismiss · x done · J jump-to-session · r refresh · q quit"
		return dimStyle.Render(help) + hidden
	}
}

func (m Model) titleWidth() int {
	w := m.width - 42
	if w < 20 {
		return 20
	}
	if w > 80 {
		return 80
	}
	return w
}

func humanizeAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
