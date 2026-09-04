package jarvis

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"fleetdm/gm/pkg/ghapi"
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
	if m.mode == modeProjectSelect && m.picker != nil {
		return m.picker.View()
	}
	switch m.state {
	case stateLoading:
		return fmt.Sprintf("\n  %s Summoning your work from GitHub…\n", m.spinner.View())
	case stateError:
		return "\n" + errStyle.Render("  Jarvis hit an error.") + "\n" +
			fmt.Sprintf("  %v\n\n", m.err) +
			dimStyle.Render("  Check `gh auth status` and your network, then press R to retry · q to quit") + "\n"
	}
	if m.branchView {
		return m.renderBranches()
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

func (m Model) focusCard(w WorkItem, selected bool) string {
	marker := "  "
	if selected {
		marker = focusStyle.Render("▸ ")
	}
	statusChip := ""
	if w.Status != "" {
		statusChip = "  " + statusStyle.Render("["+w.Status+"]")
	}
	head := fmt.Sprintf("%s#%-6d %s", marker, w.Number, truncateTitle(w.Title)) + statusChip

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
		title += " (R to refresh)"
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
	// revealLine tracks the bottom line of the item lookaheadItems positions
	// past the cursor, so scrolling keeps the next couple of items in view
	// instead of pinning the cursor to the last visible row.
	const lookaheadItems = 2
	revealLine := 0
	itemIdx := 0
	if len(m.config.PrimaryProjects) == 0 {
		lines = append(lines, m.bucketHeader(BucketPrimary, 0))
		lines = append(lines, dimStyle.Render("  use P to select projects to focus on"))
		lines = append(lines, "")
	}
	for _, bk := range BucketOrder {
		items := m.filtered.Buckets[bk]
		if len(items) == 0 {
			continue
		}
		lines = append(lines, m.bucketHeader(bk, len(items)))
		for _, it := range items {
			// Project View issues render as a two-line block (issue, then its PR/
			// branch). Everything else is a single line.
			if bk == BucketPrimary && it.Kind == KindIssue {
				lines = append(lines, m.projectIssueLines(it, itemIdx == m.cursor)...)
			} else {
				hiddenLabel := ""
				if !m.triage.Visible(m.key(it), it.Updated, now) {
					hiddenLabel = m.triage.Label(m.key(it))
				}
				lines = append(lines, m.itemLine(it, itemIdx == m.cursor, hiddenLabel, bk))
			}
			// Keep the bottom line of the cursor item and the next lookaheadItems
			// items visible when we scroll.
			if itemIdx >= m.cursor && itemIdx <= m.cursor+lookaheadItems {
				revealLine = len(lines) - 1
			}
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
		// Scroll so the look-ahead line (a couple of items past the cursor) is
		// visible; near the end of the list the clamp below lets the cursor
		// still reach the final row.
		if revealLine >= viewport {
			start = revealLine - viewport + 1
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
	title := truncateTitle(it.Title)
	marker := ""
	if it.HasSession {
		marker = " 💬"
	}
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

// projectIssueLines renders a Project View issue as a two-line block: the issue
// (number, title, project status) and, indented beneath, its linked PR + branch +
// local clone folder + PR state. The PR/branch line is omitted when there's none.
func (m Model) projectIssueLines(it Item, selected bool) []string {
	w := m.workByIssue[it.Number]
	title := truncateTitle(it.Title)

	var line1 string
	if selected {
		s := fmt.Sprintf("issue #%d  %s", it.Number, title)
		if w.Status != "" {
			s += "  [" + w.Status + "]"
		}
		line1 = selectedStyle.Render("  ▸ " + s)
	} else {
		line1 = "  " + issueTagStyle.Render(fmt.Sprintf("issue #%d", it.Number)) + "  " + title
		if w.Status != "" {
			line1 += "  " + statusStyle.Render("["+w.Status+"]")
		}
	}
	lines := []string{line1}

	if w.PR != nil || w.Branch != "" {
		var parts []string
		if w.PR != nil {
			parts = append(parts, prTagStyle.Render(fmt.Sprintf("PR #%d", w.PR.Number)))
		}
		if w.Branch != "" {
			parts = append(parts, dimStyle.Render("branch "+w.Branch))
		}
		if folder := m.branchFolder(w); folder != "" {
			parts = append(parts, dimStyle.Render("["+folder+"]"))
		}
		var pr *ghapi.PullRequest
		if w.PR != nil {
			pr = w.PR.PR
		}
		if st := prStatusLabel(pr); st != "" {
			parts = append(parts, prStatusStyle(st).Render(st))
		}
		lines = append(lines, "      "+strings.Join(parts, "  "))
	}
	return lines
}

// prStatusLabel summarizes a PR's state as draft / open / approved / merged /
// closed. "approved" means GitHub's reviewDecision is APPROVED — i.e. it has the
// approvals branch protection requires to merge; a still-open PR awaiting approval
// stays "open".
func prStatusLabel(pr *ghapi.PullRequest) string {
	if pr == nil {
		return ""
	}
	switch {
	case pr.IsMerged():
		return "merged"
	case pr.IsClosed():
		return "closed"
	case pr.IsDraft:
		return "draft"
	case pr.IsApproved():
		return "approved"
	default:
		return "open"
	}
}

// prStatusStyle colors a PR status label: grey for not-yet-actionable states
// (draft, closed), green for live/positive ones (open, approved, merged).
func prStatusStyle(label string) lipgloss.Style {
	switch label {
	case "draft", "closed":
		return dimStyle
	default:
		return reasonStyle
	}
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

// renderBranches draws the branch-cleanup view: local branches grouped by clone
// folder, each tagged with its push/merge state, plus the delete actions.
func (m Model) renderBranches() string {
	var b strings.Builder

	pushed, gone := 0, 0
	for _, r := range m.branchRows {
		switch r.State {
		case BranchPushed:
			pushed++
		case BranchGone:
			gone++
		}
	}
	title := fmt.Sprintf("🌿 Jarvis · branch cleanup · %d branch(es) · %d pushed · %d gone",
		len(m.branchRows), pushed, gone)
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

	if m.branchLoading {
		b.WriteString(dimStyle.Render("  scanning…"))
		b.WriteString("\n\n")
		b.WriteString(m.branchFooter())
		return b.String()
	}
	if len(m.branchRows) == 0 {
		b.WriteString(reasonStyle.Render("  No branches found under the configured fleet folders. 🎉"))
		b.WriteString("\n\n")
		b.WriteString(m.branchFooter())
		return b.String()
	}

	var lines []string
	cursorLine := 0
	lastRepo := ""
	for i, r := range m.branchRows {
		if r.Repo != lastRepo {
			if lastRepo != "" {
				lines = append(lines, "")
			}
			lines = append(lines, headerStyle.Render(r.Repo))
			lastRepo = r.Repo
		}
		if i == m.branchCursor {
			cursorLine = len(lines)
		}
		lines = append(lines, m.branchLine(r, i == m.branchCursor))
	}

	viewport := m.height - 7 // title + notice + blank + 2-line footer
	if viewport < 3 {
		viewport = 3
	}
	const lookahead = 2
	start := 0
	if len(lines) > viewport {
		if cursorLine+lookahead >= viewport {
			start = cursorLine + lookahead - viewport + 1
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
	b.WriteString(m.branchFooter())
	return b.String()
}

// branchLine renders one branch row: cursor marker, colored state tag, name, and
// (for protected/ahead branches) a short note on why it's kept.
func (m Model) branchLine(r BranchRow, selected bool) string {
	tag := r.State.Label()
	if r.State == BranchAhead && r.Ahead > 0 {
		tag = fmt.Sprintf("ahead %d", r.Ahead)
	}
	note := ""
	switch {
	case r.Current:
		note = "  (current — kept)"
	case r.Protected:
		note = "  (kept)"
	case r.State == BranchGone:
		note = "  (merged & remote gone — d to delete)"
	}

	if selected {
		plain := fmt.Sprintf("  ▸ [%-8s] %s%s", tag, r.Branch, note)
		return selectedStyle.Render(plain)
	}
	styledTag := branchStateStyle(r.State).Render(fmt.Sprintf("[%-8s]", tag))
	return "    " + styledTag + " " + r.Branch + dimStyle.Render(note)
}

// branchStateStyle colors a branch's state tag: green for safely-pushed, orange
// for ahead (unpushed work), red for gone, grey for never-pushed local-only.
func branchStateStyle(s BranchState) lipgloss.Style {
	switch s {
	case BranchPushed:
		return reasonStyle
	case BranchAhead:
		return statusStyle
	case BranchGone:
		return errStyle
	default:
		return dimStyle
	}
}

func (m Model) branchFooter() string {
	if m.mode == modeConfirmBranchDelete {
		return errStyle.Render(m.branchPlanMsg+" ") + "[y] yes  " + dimStyle.Render("· any other key cancels")
	}
	nav := "↑/↓ move · g/G top/bottom · F fetch+prune · r rescan · B/esc back · q quit"
	actions := "d delete selected · p delete all pushed · D delete all but main"
	return dimStyle.Render(nav) + "\n" + dimStyle.Render(actions)
}

func (m Model) footer() string {
	switch m.mode {
	case modeSnooze:
		return dimStyle.Render("snooze: ") +
			"[1] 1 hour  [2] 4 hours  [3] tomorrow  [4] 1 week  " +
			dimStyle.Render("· esc cancel")
	case modeConfirmMerge:
		n := ""
		if prItem, _, _, ok := m.mergeTarget(); ok {
			n = fmt.Sprintf("#%d", prItem.Number)
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
	case modeNewClone:
		return m.newCloneFooter()
	default:
		if m.focusView {
			nav := "↑/↓ move · g/G top/bottom · enter open · J jump · b project · f board-view · r/R refresh(one/all) · q quit"
			actions := "w start · v in-review · m merge · M merge+cherry-pick · a awaiting-qa · p unpin"
			if m.config.EffectiveRole() == RoleQA {
				actions += " · t test"
			}
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
		nav := "↑/↓ move · g/G top/bottom · enter open · b project · f focus · B branches · J jump · r/R refresh(one/all) · q quit"
		actions := "w start · v review · m merge · M merge+cp · p pin · P projects · c comment · s snooze · d dismiss · x done · u clear"
		if m.config.EffectiveRole() == RoleQA {
			actions += " · t test"
		}
		return dimStyle.Render(nav) + hidden + "\n" + dimStyle.Render(actions)
	}
}

// startWorkFooter renders the branch input and the clone picker. In QA "test"
// mode there's no branch to name — it's just a clone picker for the repro session.
func (m Model) startWorkFooter() string {
	var b strings.Builder
	if m.startQA {
		b.WriteString(titleBarStyle.Render(fmt.Sprintf("Test #%d", m.startIssue)))
		b.WriteString("\n")
	} else {
		b.WriteString(titleBarStyle.Render(fmt.Sprintf("Start work on #%d", m.startIssue)))
		b.WriteString("\n")
		b.WriteString("branch: " + m.startBranchInput.View() + "\n")
	}
	// QA needs an existing clone to test in; a fresh clone has nothing to verify.
	if len(m.startClones) == 0 && m.startQA {
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
	// Non-QA: a virtual row to clone a fresh fleet-* working dir.
	if !m.startQA {
		newRow := "＋ create new fleet-… working dir"
		if m.startCloneCursor == len(m.startClones) {
			b.WriteString(selectedStyle.Render("▸ "+newRow) + "\n")
		} else {
			b.WriteString("  " + reasonStyle.Render(newRow) + "\n")
		}
	}
	action := "enter start · esc cancel"
	if m.startQA {
		action = "enter test · esc cancel"
	}
	b.WriteString(dimStyle.Render(action))
	return b.String()
}

func (m Model) newCloneFooter() string {
	var b strings.Builder
	b.WriteString(titleBarStyle.Render(fmt.Sprintf("New working dir · start #%d on %s", m.startIssue, m.startBranch)))
	b.WriteString("\n")
	base := "~/projects"
	if len(m.config.CloneBaseDirs) > 0 {
		base = m.config.CloneBaseDirs[0]
	}
	b.WriteString(dimStyle.Render("clones "+m.repo+" into "+base+"/") + "fleet-" + m.newCloneInput.View() + "\n")
	b.WriteString(dimStyle.Render("enter clone & start · esc back to clone list"))
	return b.String()
}

func truncateTitle(s string) string {
	r := []rune(s)
	if len(r) <= 35 {
		return s
	}
	return string(r[:32]) + "..."
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
