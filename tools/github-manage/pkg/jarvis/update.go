package jarvis

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case fetchDoneMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.applyFetch(msg.res)
		m.lastRefresh = time.Now()
		m.fromCache = false
		m.state = stateLoaded
		_ = SaveCache(m.cachePath, msg.res) // keep the cache warm for the next open
		return m, nil

	case actionResultMsg:
		if msg.err != nil {
			m.notice = msg.verb + " failed: " + truncate(firstLine(msg.out), 80)
			m.noticeErr = true
			return m, nil
		}
		m.notice = msg.verb + " ✓"
		m.noticeErr = false
		if msg.verb == "merge" && msg.key != "" {
			// A merged PR is done; hide it locally so it clears immediately.
			m.triage.Done(msg.key, time.Now())
			_ = m.triage.Save()
			m.rebuild()
			// Merge + cherry-pick: hand off to a Claude session for the cherry-pick.
			if msg.cherryPick {
				m.notice = "merged ✓ — starting cherry-pick session"
				return m, launchSessionCmd(msg.clonePath, cherryPickPrompt(msg.pr))
			}
			// Otherwise advance the linked issue to Awaiting QA (drops it from focus).
			if msg.issue != 0 && msg.project != 0 {
				return m, setStatusCmd(msg.issue, msg.project, statusAwaitingQA)
			}
		}
		return m, nil

	case startWorkDoneMsg:
		if msg.err != nil {
			m.notice = "start work failed: " + truncate(firstLine(msg.err.Error()), 90)
			m.noticeErr = true
			return m, nil
		}
		// Record the authoritative link, auto-pin, and reflect the new status now.
		m.links.Set(msg.issue, Link{ClonePath: msg.clonePath, Branch: msg.branch, Project: msg.project})
		_ = m.links.Save()
		m.focus.Add(msg.issue)
		_ = m.focus.Save()
		if msg.statusSet != "" {
			m.statuses[msg.issue] = msg.statusSet
			if msg.project != 0 {
				m.projects[msg.issue] = msg.project
			}
		}
		m.rebuild()
		m.notice = fmt.Sprintf("started #%d on %s ✓", msg.issue, msg.branch)
		m.noticeErr = false
		if msg.warn != "" {
			m.notice += " · " + msg.warn
		}
		// Launch a fresh Claude session in the new clone/branch, seeded with context.
		return m, launchSessionCmd(msg.clonePath, m.startPrompt(msg.issue))

	case statusWriteMsg:
		if msg.err != nil {
			m.notice = "status update failed: " + truncate(firstLine(msg.err.Error()), 90)
			m.noticeErr = true
			return m, nil
		}
		m.statuses[msg.issue] = msg.statusSet
		// Awaiting QA / closed work drops out of focus automatically.
		if statusHas(msg.statusSet, "await") || statusHas(msg.statusSet, "qa") {
			m.focus.Remove(msg.issue)
			_ = m.focus.Save()
		}
		m.rebuild()
		m.notice = fmt.Sprintf("#%d → %s ✓", msg.issue, msg.statusSet)
		m.noticeErr = false
		return m, nil

	case itemRefreshedMsg:
		if msg.err != nil {
			m.notice = fmt.Sprintf("refresh #%d failed: %s", msg.number, truncate(firstLine(msg.err.Error()), 80))
			m.noticeErr = true
			return m, nil
		}
		switch msg.kind {
		case KindPR:
			if msg.pr != nil {
				m.replacePR(*msg.pr)
			}
		case KindIssue:
			m.statuses[msg.number] = msg.status
			if msg.project != 0 {
				m.projects[msg.number] = msg.project
			}
			m.issueProjects[msg.number] = msg.refs
			if msg.closed {
				// Closed on GitHub → done; hide it.
				m.triage.Done(m.key(Item{Kind: KindIssue, Number: msg.number}), time.Now())
				_ = m.triage.Save()
			}
		}
		m.autoMarkCompleted()
		m.rebuild()
		_ = SaveCache(m.cachePath, m.currentFetchResult())
		m.notice = fmt.Sprintf("refreshed #%d ✓", msg.number)
		m.noticeErr = false
		return m, nil

	case sessionReturnedMsg:
		// Returned from a Claude session jarvis launched/resumed. Don't hit GitHub —
		// just drop back to the cached loadout we already have in memory (press r to
		// refresh when you actually want fresh data).
		m.state = stateLoaded
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.state == stateLoading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.mode == modeComment {
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	}
	if m.mode == modeStartWork {
		var cmd tea.Cmd
		m.startBranchInput, cmd = m.startBranchInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeSnooze:
		return m.handleSnoozeKey(msg)
	case modeConfirmMerge:
		return m.handleConfirmMergeKey(msg)
	case modeComment:
		return m.handleCommentKey(msg)
	case modeStartWork:
		return m.handleStartWorkKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = "" // any keypress clears a stale notice
	switch msg.String() {
	case "esc":
		if m.focusView {
			m.focusView = false
			return m, nil
		}
		return m, tea.Quit
	case "q", "ctrl+c":
		return m, tea.Quit

	case "r":
		m.state = stateLoading
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

	case "R":
		// Small refresh: re-fetch only the highlighted item's data (no full pull).
		if m.focusView {
			if w, ok := m.currentWork(); ok {
				m.notice = fmt.Sprintf("refreshing #%d…", w.Number)
				cmds := []tea.Cmd{refreshIssueCmd(m.repo, w.Number, w.Project)}
				if w.PR != nil {
					cmds = append(cmds, refreshPRCmd(m.repo, w.PR.Number))
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}
		if it, ok := m.currentItem(); ok {
			switch it.Kind {
			case KindPR:
				m.notice = fmt.Sprintf("refreshing PR #%d…", it.Number)
				return m, refreshPRCmd(m.repo, it.Number)
			case KindIssue:
				m.notice = fmt.Sprintf("refreshing #%d…", it.Number)
				return m, refreshIssueCmd(m.repo, it.Number, m.workByIssue[it.Number].Project)
			default:
				m.notice = "nothing to refresh here (r for a full refresh)"
			}
		}

	case "up", "k":
		if m.focusView {
			if m.focusCursor > 0 {
				m.focusCursor--
			}
		} else if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.focusView {
			if m.focusCursor < len(m.focusList)-1 {
				m.focusCursor++
			}
		} else if m.cursor < len(m.flat)-1 {
			m.cursor++
		}
	case "g", "home":
		if m.focusView {
			m.focusCursor = 0
		} else {
			m.cursor = 0
		}
	case "G", "end":
		if m.focusView {
			m.focusCursor = len(m.focusList) - 1
		} else if len(m.flat) > 0 {
			m.cursor = len(m.flat) - 1
		}

	case "f":
		// Toggle the focus (pinned work items) view.
		m.focusView = !m.focusView

	case "b":
		// On a project header row, open that project board.
		if it, ok := m.currentItem(); ok && it.Kind == KindProject {
			if it.URL != "" {
				return m, openURLCmd(it.URL)
			}
			m.notice = "no URL for this project"
			m.noticeErr = true
			return m, nil
		}
		// Otherwise open the selected issue's most recently updated project board.
		if w, ok := m.currentWork(); ok {
			if num := m.mostRecentProject(w.Number); num != 0 {
				m.notice = fmt.Sprintf("opening #%d's latest project (#%d)", w.Number, num)
				return m, openURLCmd(m.orgProjectURL(num))
			}
			m.notice = fmt.Sprintf("#%d isn't on any known project (try r to refresh)", w.Number)
			m.noticeErr = true
			return m, nil
		}
		m.notice = "select an issue first"
		m.noticeErr = true

	case "H":
		m.showHidden = !m.showHidden
		m.rebuild()

	case "enter", "o":
		if m.focusView {
			if w, ok := m.currentWork(); ok {
				return m, openURLCmd(w.URL)
			}
			return m, nil
		}
		if it, ok := m.currentItem(); ok {
			if it.Kind == KindSession {
				return m, resumeSessionCmd(it.SessionID, it.Cwd)
			}
			return m, openURLCmd(it.URL)
		}

	case "J":
		// Jump into the Claude session driving the selected work/item.
		if m.focusView {
			if w, ok := m.currentWork(); ok && w.SessionID != "" {
				return m, resumeSessionCmd(w.SessionID, w.Cwd)
			}
			return m, nil
		}
		if it, ok := m.currentItem(); ok && (it.HasSession || it.Kind == KindSession) {
			return m, resumeSessionCmd(it.SessionID, it.Cwd)
		}

	case "p":
		// Pin/unpin the selected work item to/from focus.
		if w, ok := m.currentWork(); ok {
			on := m.focus.Toggle(w.Number)
			_ = m.focus.Save()
			if on {
				m.notice = fmt.Sprintf("pinned #%d", w.Number)
			} else {
				m.notice = fmt.Sprintf("unpinned #%d", w.Number)
			}
			m.rebuild()
		}

	case "v":
		// Mark the selected work item's issue In review.
		if w, ok := m.currentWork(); ok {
			if w.Project == 0 {
				m.notice = fmt.Sprintf("no project board known for #%d", w.Number)
				m.noticeErr = true
				return m, nil
			}
			m.notice = "updating status…"
			return m, setStatusCmd(w.Number, w.Project, statusInReview)
		}

	case "a":
		// Mark the selected work item's issue Awaiting QA (e.g. PR merged elsewhere).
		if w, ok := m.currentWork(); ok {
			if w.Project == 0 {
				m.notice = fmt.Sprintf("no project board known for #%d", w.Number)
				m.noticeErr = true
				return m, nil
			}
			m.notice = "updating status…"
			return m, setStatusCmd(w.Number, w.Project, statusAwaitingQA)
		}

	case "s":
		if it, ok := m.currentItem(); ok && it.Kind != KindProject {
			m.mode = modeSnooze
		}
	case "d":
		if it, ok := m.currentItem(); ok && it.Kind != KindProject {
			m.triage.Dismiss(m.key(it), it.Updated)
			_ = m.triage.Save()
			m.notice = "dismissed"
			m.rebuild()
		}
	case "x":
		if it, ok := m.currentItem(); ok && it.Kind != KindProject {
			m.triage.Done(m.key(it), it.Updated)
			_ = m.triage.Save()
			m.notice = "marked done"
			m.rebuild()
		}
	case "u":
		if it, ok := m.currentItem(); ok && it.Kind != KindProject {
			m.triage.Clear(m.key(it))
			_ = m.triage.Save()
			m.notice = "cleared"
			m.rebuild()
		}

	case "w":
		// Start work on the selected issue: name a branch, pick a clone.
		if w, ok := m.currentWork(); ok {
			m.startIssue = w.Number
			m.startProject = w.Project
			m.startBranchInput.SetValue(suggestBranch(m.login, w.Number, w.Title))
			m.startBranchInput.CursorEnd()
			m.startBranchInput.Focus()
			m.startClones = DiscoverClones(m.config.CloneBaseDirs, m.repo)
			m.startCloneCursor = 0
			m.mode = modeStartWork
			return m, nil
		}

	case "m":
		// Merge: a PR item in board view, or the selected work item's PR in focus view.
		if m.focusView {
			if w, ok := m.currentWork(); ok && w.PR != nil {
				m.mode = modeConfirmMerge
			}
		} else if it, ok := m.currentItem(); ok && it.Kind == KindPR {
			m.mode = modeConfirmMerge
		}

	case "M":
		// Merge, then start a Claude cherry-pick session for the merged PR.
		if m.focusView {
			if w, ok := m.currentWork(); ok && w.PR != nil {
				m.mergeCherryPick = true
				m.mode = modeConfirmMerge
			}
		} else if it, ok := m.currentItem(); ok && it.Kind == KindPR {
			m.mergeCherryPick = true
			m.mode = modeConfirmMerge
		}
	case "c":
		if _, ok := m.currentItem(); ok {
			m.mode = modeComment
			m.commentInput.SetValue("")
			m.commentInput.Focus()
			return m, nil
		}
	}
	return m, nil
}

// snoozeOptions maps the picker keys to durations.
var snoozeOptions = []struct {
	key   string
	label string
	dur   time.Duration
}{
	{"1", "1 hour", time.Hour},
	{"2", "4 hours", 4 * time.Hour},
	{"3", "tomorrow", 24 * time.Hour},
	{"4", "1 week", 7 * 24 * time.Hour},
}

func (m *Model) handleSnoozeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeNormal
		return m, nil
	}
	for _, opt := range snoozeOptions {
		if msg.String() == opt.key {
			if it, ok := m.currentItem(); ok {
				m.triage.Snooze(m.key(it), time.Now().Add(opt.dur), it.Updated)
				_ = m.triage.Save()
				m.notice = "snoozed " + opt.label
				m.rebuild()
			}
			m.mode = modeNormal
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) handleConfirmMergeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		cp := m.mergeCherryPick
		m.mergeCherryPick = false
		prItem, issue, project, ok := m.mergeTarget()
		if !ok {
			return m, nil
		}
		m.notice = "merging…"
		if cp {
			return m, mergeCherryPickCmd(m.repo, prItem, m.key(prItem), m.cherryPickClone(issue))
		}
		return m, mergeCmd(m.repo, prItem, m.key(prItem), issue, project)
	default:
		m.mode = modeNormal
		m.mergeCherryPick = false
	}
	return m, nil
}

// mergeTarget resolves the PR to merge and its linked issue/project (for the
// Awaiting QA follow-up), in both the board and focus views.
func (m *Model) mergeTarget() (prItem Item, issue, project int, ok bool) {
	if m.focusView {
		if w, wok := m.currentWork(); wok && w.PR != nil {
			return *w.PR, w.Number, w.Project, true
		}
		return Item{}, 0, 0, false
	}
	it, iok := m.currentItem()
	if !iok || it.Kind != KindPR {
		return Item{}, 0, 0, false
	}
	if w, wok := m.workForPR(it.Number); wok {
		return it, w.Number, w.Project, true
	}
	return it, 0, 0, true
}

func (m *Model) handleCommentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.commentInput.Blur()
		return m, nil
	case "enter":
		body := m.commentInput.Value()
		m.mode = modeNormal
		m.commentInput.Blur()
		if it, ok := m.currentItem(); ok && body != "" {
			m.notice = "posting comment…"
			return m, commentCmd(m.repo, it, body)
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	}
}

func (m *Model) handleStartWorkKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.startBranchInput.Blur()
		return m, nil
	case "up", "ctrl+p":
		if m.startCloneCursor > 0 {
			m.startCloneCursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.startCloneCursor < len(m.startClones)-1 {
			m.startCloneCursor++
		}
		return m, nil
	case "enter":
		branch := strings.TrimSpace(m.startBranchInput.Value())
		if branch == "" {
			m.notice = "enter a branch name"
			m.noticeErr = true
			return m, nil
		}
		if len(m.startClones) == 0 {
			m.notice = "no local clone of " + m.repo + " found (set clone_base_dirs in config.json)"
			m.noticeErr = true
			m.mode = modeNormal
			return m, nil
		}
		clone := m.startClones[m.startCloneCursor]
		m.mode = modeNormal
		m.startBranchInput.Blur()
		m.notice = "starting work…"
		m.noticeErr = false
		return m, startWorkCmd(m.startIssue, m.startProject, clone.Path, branch)
	default:
		var cmd tea.Cmd
		m.startBranchInput, cmd = m.startBranchInput.Update(msg)
		return m, cmd
	}
}

// startPrompt builds the seed prompt for a freshly launched Claude session.
func (m *Model) startPrompt(issue int) string {
	w, ok := m.workByIssue[issue]
	if !ok {
		return fmt.Sprintf("Let's work on issue #%d.", issue)
	}
	prompt := fmt.Sprintf("Let's work on issue #%d: %s", issue, w.Title)
	if w.URL != "" {
		prompt += "\n" + w.URL
	}
	return prompt
}

func (m *Model) currentItem() (Item, bool) {
	if m.cursor >= 0 && m.cursor < len(m.flat) {
		return m.flat[m.cursor], true
	}
	return Item{}, false
}
