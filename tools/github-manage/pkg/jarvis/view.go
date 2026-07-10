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
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F0883E"))
	focusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
	projectStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#39C5CF"))
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
	if m.focusView {
		return m.renderFocus()
	}
	return m.renderBoard()
}

// renderFocus renders the pinned work items as issue-centric cards: issue +
// project status + linked PR + the single most useful next action.
func (m Model) renderFocus() string {
	var b strings.Builder
	title := fmt.Sprintf("🎩 Jarvis · Focus · %d pinned", len(m.focusList))
	b.WriteString(titleBarStyle.Render(title))
	b.WriteString("\n")
	if m.notice != "" {
		style := noticeStyle
		if m.noticeErr {
			style = errStyle
		}
		b.WriteString(style.Render("  "+m.notice) + "\n")
	}
	b.WriteString("\n")

	if len(m.focusList) == 0 {
		b.WriteString(dimStyle.Render("  No focused work. Press p on an issue to pin it; f to return to the board."))
		b.WriteString("\n\n")
		b.WriteString(m.footer())
		return b.String()
	}

	for i, w := range m.focusList {
		b.WriteString(m.focusCard(w, i == m.focusCursor))
		b.WriteString("\n")
	}
	b.WriteString(m.footer())
	return b.String()
}

// focusCard renders one work item as a multi-line card.
func (m Model) focusCard(w WorkItem, selected bool) string {
	marker := "  "
	if selected {
		marker = focusStyle.Render("▸ ")
	}
	statusChip := ""
	if w.Status != "" {
		statusChip = "  " + statusStyle.Render("["+w.Status+"]")
	}
	head := fmt.Sprintf("%s#%-6d %s", marker, w.Number, truncate(w.Title, m.titleWidth())) + statusChip

	// PR / session line.
	var detail string
	switch {
	case w.PR != nil:
		detail = prTagStyle.Render(fmt.Sprintf("PR #%d", w.PR.Number)) + "  " + reasonStyle.Render(w.PR.Reason)
	case w.SessionID != "":
		detail = sessTagStyle.Render("session active, no PR yet")
	default:
		detail = dimStyle.Render("no branch/PR yet")
	}
	if w.SessionID != "" {
		detail += "  💬"
	}
	if w.Branch != "" {
		detail += dimStyle.Render("  [" + w.Branch + "]")
	}

	// Next-action line.
	next := ""
	if w.Next != ActNone {
		next = "     " + reasonStyle.Render("▸ next: "+w.Next.Label()) + dimStyle.Render(" ("+w.Next.Key()+")")
	}

	lines := []string{head, "     " + detail}
	if next != "" {
		lines = append(lines, next)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderBoard() string {
	var b strings.Builder

	when := "refreshed"
	if m.fromCache {
		when = "cached"
	}
	title := fmt.Sprintf("🎩 Jarvis · @%s · %d shown · %s %s",
		m.login, m.filtered.Total(), when, m.lastRefresh.Format("15:04:05"))
	if m.fromCache {
		title += " (r to refresh)"
	}
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
			lines = append(lines, m.itemLine(it, itemIdx == m.cursor, hiddenLabel, bk))
			itemIdx++
		}
		lines = append(lines, "")
	}

	viewport := m.height - 7 // title + notice + blank + 2-line footer
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

func (m Model) itemLine(it Item, selected bool, hiddenLabel string, bk Bucket) string {
	if it.Kind == KindProject {
		return m.projectLine(it, selected)
	}
	// Project View issues sit indented under their project header.
	indent := ""
	if bk == BucketPrimary && it.Kind == KindIssue {
		indent = "  "
	}
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
	// Issue-centric annotation: project status + linked PR, from the work overlay.
	statusText, prText, focused := m.issueAnnotation(it)
	focusMark := ""
	if focused {
		focusMark = "★ "
	}
	reason := it.Reason
	if hiddenLabel != "" {
		reason = hiddenLabel
	}

	if selected {
		annot := ""
		if statusText != "" {
			annot += "  [" + statusText + "]"
		}
		annot += prText
		plain := fmt.Sprintf("▸ %s%s%-6s %-7s %s%s   %s%s   %s", indent, focusMark, kind, num, title, marker, reason, annot, age)
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
	annot := ""
	if statusText != "" {
		annot += "  " + statusStyle.Render("["+statusText+"]")
	}
	if prText != "" {
		annot += prTagStyle.Render(prText)
	}
	return fmt.Sprintf("  %s%s%s %-7s %s%s   %s%s   %s",
		indent, focusStyle.Render(focusMark), styledLabel, num, title, marker, reasonStyled, annot, dimStyle.Render(age))
}

// projectLine renders a KindProject header row (always shown, navigable, opened
// with b/enter).
func (m Model) projectLine(it Item, selected bool) string {
	body := fmt.Sprintf("%s   %s   %s", it.Title, it.Reason, "b/enter to open")
	if selected {
		return selectedStyle.Render("▸ " + body)
	}
	return "  " + projectStyle.Render(it.Title) + "   " + statusStyle.Render(it.Reason) + "   " + dimStyle.Render("b/enter to open")
}

// issueAnnotation returns the project status and linked-PR text for an issue item
// (from the work overlay), plus whether it's focused. Non-issues return zeros.
func (m Model) issueAnnotation(it Item) (status, prText string, focused bool) {
	if it.Kind != KindIssue {
		return "", "", false
	}
	w, ok := m.workByIssue[it.Number]
	if !ok {
		return "", "", false
	}
	if w.PR != nil {
		prText = fmt.Sprintf(" → PR #%d", w.PR.Number)
	}
	return w.Status, prText, w.Focused
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
		if m.focusView {
			if w, ok := m.currentWork(); ok && w.PR != nil {
				n = fmt.Sprintf("#%d", w.PR.Number)
			}
		}
		prompt := fmt.Sprintf("Merge %s with squash? ", n)
		if m.mergeCherryPick {
			prompt = fmt.Sprintf("Merge %s (squash) + start cherry-pick session? ", n)
		}
		return errStyle.Render(prompt) +
			"[y] yes  " + dimStyle.Render("· any other key cancels")
	case modeComment:
		return m.commentInput.View() + "\n" + dimStyle.Render("enter post · esc cancel")
	case modeStartWork:
		return m.startWorkFooter()
	default:
		if m.focusView {
			nav := "↑/↓ move · g/G top/bottom · enter open · J jump · b project · f board-view · r/R refresh(all/one) · q quit"
			actions := "w start · v in-review · m merge · M merge+cherry-pick · a awaiting-qa · p unpin"
			return dimStyle.Render(nav) + "\n" + dimStyle.Render(actions)
		}
		hidden := ""
		if m.hidden > 0 {
			state := "show"
			if m.showHidden {
				state = "hide"
			}
			hidden = dimStyle.Render(fmt.Sprintf(" · %d hidden (H to %s)", m.hidden, state))
		}
		nav := "↑/↓ move · g/G top/bottom · enter open · b project · f focus · J jump · r/R refresh(all/one) · q quit"
		actions := "w start · v review · m merge · M merge+cp · p pin · c comment · s snooze · d dismiss · x done · u clear"
		return dimStyle.Render(nav) + hidden + "\n" + dimStyle.Render(actions)
	}
}

// startWorkFooter renders the branch input and the clone picker.
func (m Model) startWorkFooter() string {
	var b strings.Builder
	b.WriteString(titleBarStyle.Render(fmt.Sprintf("Start work on #%d", m.startIssue)))
	b.WriteString("\n")
	b.WriteString("branch: " + m.startBranchInput.View() + "\n")
	if len(m.startClones) == 0 {
		b.WriteString(errStyle.Render(fmt.Sprintf("  no local clone of %s under %s", m.repo, strings.Join(m.config.CloneBaseDirs, ", "))))
		b.WriteString("\n" + dimStyle.Render("esc cancel · set clone_base_dirs in ~/.config/gm/jarvis/config.json"))
		return b.String()
	}
	b.WriteString(dimStyle.Render("pick clone (↑/↓):") + "\n")
	for i, c := range m.startClones {
		state := dimStyle.Render("busy: " + c.Branch)
		if c.Free() {
			state = reasonStyle.Render("free")
		} else if !c.Clean {
			state = statusStyle.Render(c.Branch + " · dirty")
		}
		line := fmt.Sprintf("  %s  %s", c.Path, state)
		if i == m.startCloneCursor {
			line = selectedStyle.Render("▸ "+fmt.Sprintf("%s  ", c.Path)) + " " + state
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(dimStyle.Render("enter start · esc cancel"))
	return b.String()
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
