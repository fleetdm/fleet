package jarvis

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"fleetdm/gm/pkg/ghapi"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// The project picker (P) owns all input while open.
	if m.mode == modeProjectSelect && m.picker != nil {
		return m.updateProjectSelect(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case projectPickerReadyMsg:
		if msg.err != nil {
			m.notice = "couldn't load projects: " + truncate(firstLine(msg.err.Error()), 80)
			m.noticeErr = true
			return m, nil
		}
		open := msg.projects[:0]
		for _, p := range msg.projects {
			if !p.Closed {
				open = append(open, p)
			}
		}
		if len(open) == 0 {
			m.notice = "no open projects to choose from"
			m.noticeErr = true
			return m, nil
		}
		pk := newOnboardModel(repoOwner(m.repo), open)
		pk.embedded = true
		pk.seedSelection(m.config.PrimaryProjects)
		m.picker = pk
		m.mode = modeProjectSelect
		m.notice = ""
		return m, pk.Init()

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
		// SetAndSave merges with disk so a sibling jarvis instance can see this branch
		// is started locally (and we don't clobber links it wrote).
		_ = m.links.SetAndSave(msg.issue, Link{ClonePath: msg.clonePath, Branch: msg.branch, Project: msg.project})
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
				switch {
				case msg.forIssue != 0 && msg.pr.IsDone():
					// Merged/closed PR found by branch — record it against the issue
					// (kept off the board) so it shows as merged, ready for QA.
					if m.mergedPRs == nil {
						m.mergedPRs = map[int]*ghapi.PullRequest{}
					}
					m.mergedPRs[msg.forIssue] = msg.pr
				default:
					m.replacePR(*msg.pr)
				}
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

	case projectRefreshedMsg:
		if msg.err != nil {
			m.notice = fmt.Sprintf("refresh project failed: %s", truncate(firstLine(msg.err.Error()), 80))
			m.noticeErr = true
			return m, nil
		}
		m.replaceProjectView(msg)
		m.autoMarkCompleted()
		m.rebuild()
		_ = SaveCache(m.cachePath, m.currentFetchResult())
		m.notice = fmt.Sprintf("refreshed %s ✓", msg.title)
		m.noticeErr = false
		return m, nil

	case branchScanMsg:
		m.branchRows = msg.rows
		m.branchLoading = false
		if m.branchCursor >= len(m.branchRows) {
			m.branchCursor = len(m.branchRows) - 1
		}
		if m.branchCursor < 0 {
			m.branchCursor = 0
		}
		if msg.pruned {
			m.notice = "fetched + pruned · rescanned"
			m.noticeErr = false
		}
		return m, nil

	case branchDeleteDoneMsg:
		if msg.failed > 0 {
			m.notice = fmt.Sprintf("deleted %d branch(es) · %d failed", msg.deleted, msg.failed)
			m.noticeErr = true
		} else {
			m.notice = fmt.Sprintf("deleted %d branch(es)", msg.deleted)
			m.noticeErr = false
		}
		// Rescan (offline) so the view reflects the deletions immediately.
		m.branchLoading = true
		return m, scanBranchesCmd(m.config.CloneBaseDirs, m.config.BranchScanGlob, false)

	case sessionReturnedMsg:
		// Returned from a Claude session jarvis launched/resumed. Don't hit GitHub —
		// just drop back to the cached loadout we already have in memory (press R to
		// refresh everything when you actually want fresh data).
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
	if m.mode == modeNewClone {
		var cmd tea.Cmd
		m.newCloneInput, cmd = m.newCloneInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateProjectSelect drives the in-dashboard project picker. It delegates every
// message to the embedded picker, then acts on the outcome: on confirm it saves
// the selection to config and triggers a full refresh so the Project View reflects
// it; on cancel it returns to the board unchanged.
func (m *Model) updateProjectSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = ws.Width, ws.Height
	}
	updated, cmd := m.picker.Update(msg)
	m.picker = updated.(*onboardModel)

	switch {
	case m.picker.confirmed:
		handles := m.picker.chosenHandles()
		m.picker = nil
		m.mode = modeNormal
		m.config.PrimaryProjects = handles
		if err := m.config.Save(DefaultConfigPath()); err != nil {
			m.notice = "saving projects failed: " + truncate(firstLine(err.Error()), 80)
			m.noticeErr = true
			return m, nil
		}
		m.notice = fmt.Sprintf("saved %d project(s) · refreshing…", len(handles))
		m.noticeErr = false
		m.state = stateLoading
		return m, tea.Batch(m.spinner.Tick, m.fetchCmd())
	case m.picker.cancelled:
		m.picker = nil
		m.mode = modeNormal
		m.notice = "project selection cancelled"
		return m, nil
	}
	return m, cmd
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
	case modeConfirmBranchDelete:
		return m.handleConfirmBranchDeleteKey(msg)
	case modeNewClone:
		return m.handleNewCloneKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = "" // any keypress clears a stale notice
	// The branch-cleanup view owns navigation and delete keys while open.
	if m.branchView {
		return m.handleBranchKey(msg)
	}
	switch msg.String() {
	case "esc":
		if m.focusView {
			m.focusView = false
			return m, nil
		}
		return m, tea.Quit
	case "q", "ctrl+c":
		return m, tea.Quit

	case "R":
		m.state = stateLoading
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

	case "r":
		// Small refresh: re-fetch only the highlighted item's data (no full pull).
		if m.focusView {
			if w, ok := m.currentWork(); ok {
				m.notice = fmt.Sprintf("refreshing #%d…", w.Number)
				cmds := []tea.Cmd{refreshIssueCmd(m.repo, w.Number, w.Project)}
				switch {
				case w.PR != nil:
					cmds = append(cmds, refreshPRCmd(m.repo, w.PR.Number))
				case w.Branch != "":
					// No PR linked yet — look for one opened/merged since the last full fetch.
					cmds = append(cmds, refreshPRByBranchCmd(m.repo, w.Branch, w.Number))
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
				w := m.workByIssue[it.Number]
				cmds := []tea.Cmd{refreshIssueCmd(m.repo, it.Number, w.Project)}
				switch {
				case w.PR != nil:
					// Re-fetch the linked PR so its draft/approval/CI state updates too.
					cmds = append(cmds, refreshPRCmd(m.repo, w.PR.Number))
				case w.Branch != "":
					// No PR linked yet but we know the branch — discover one opened
					// since the last full fetch and inject it into the board.
					cmds = append(cmds, refreshPRByBranchCmd(m.repo, w.Branch, it.Number))
				}
				return m, tea.Batch(cmds...)
			case KindProject:
				m.notice = fmt.Sprintf("refreshing project %s…", it.Title)
				return m, refreshProjectCmd(m.repo, it.Number, m.login, m.config.EffectiveRole(), m.linkBranches())
			default:
				m.notice = "nothing to refresh here (R for a full refresh)"
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

	case "B":
		// Open the branch-cleanup view: scan the configured fleet folders offline.
		m.branchView = true
		m.branchCursor = 0
		m.branchLoading = true
		m.notice = ""
		return m, scanBranchesCmd(m.config.CloneBaseDirs, m.config.BranchScanGlob, false)

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
			m.notice = fmt.Sprintf("#%d isn't on any known project (try R to refresh)", w.Number)
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

	case "P":
		// Re-open the project picker to change which boards drive the Project View.
		m.notice = "loading projects…"
		m.noticeErr = false
		return m, projectPickerCmd(m.repo)

	case "v":
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
			m.startQA = false
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

	case "t":
		// QA: launch a repro/verify Claude session on the selected item — pick a
		// clone; no branch or status change (the QA prompt drives the git setup).
		if w, ok := m.currentWork(); ok && w.Next == ActTestQA {
			m.startQA = true
			m.startIssue = w.Number
			m.startProject = w.Project
			m.startBranchInput.Blur()
			m.startClones = DiscoverClones(m.config.CloneBaseDirs, m.repo)
			m.startCloneCursor = 0
			m.mode = modeStartWork
			return m, nil
		}

	case "m":
		// Merge the selected item's PR (a Project View/focus issue's linked PR, or a
		// standalone PR row), then advance the linked issue to Awaiting QA.
		if _, _, _, ok := m.mergeTarget(); ok {
			m.mode = modeConfirmMerge
		} else {
			m.notice = "no PR to merge here"
		}

	case "M":
		// Merge, then start a Claude cherry-pick session for the merged PR.
		if _, _, _, ok := m.mergeTarget(); ok {
			m.mergeCherryPick = true
			m.mode = modeConfirmMerge
		} else {
			m.notice = "no PR to merge here"
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

// handleBranchKey drives the branch-cleanup view: navigation, a fetch+prune
// rescan (F), and the three delete flows (d individual, p all-pushed, D
// all-but-main), each of which routes through a y/N confirmation.
func (m *Model) handleBranchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "B":
		m.branchView = false
		return m, nil

	case "up", "k":
		if m.branchCursor > 0 {
			m.branchCursor--
		}
	case "down", "j":
		if m.branchCursor < len(m.branchRows)-1 {
			m.branchCursor++
		}
	case "g", "home":
		m.branchCursor = 0
	case "G", "end":
		if len(m.branchRows) > 0 {
			m.branchCursor = len(m.branchRows) - 1
		}

	case "r", "R":
		m.branchLoading = true
		return m, scanBranchesCmd(m.config.CloneBaseDirs, m.config.BranchScanGlob, false)

	case "F":
		// Fetch + prune each repo so merged-then-deleted branches show as gone.
		m.branchLoading = true
		m.notice = "fetching + pruning…"
		m.noticeErr = false
		return m, scanBranchesCmd(m.config.CloneBaseDirs, m.config.BranchScanGlob, true)

	case "d", "x":
		// Delete just the highlighted branch (handles the merged-from-web, no
		// upstream case the bulk sweeps miss).
		if r, ok := m.currentBranch(); ok {
			if r.Protected {
				m.notice = fmt.Sprintf("can't delete %s (protected)", r.Branch)
				m.noticeErr = true
				return m, nil
			}
			m.branchPlan = []BranchRow{r}
			m.branchPlanMsg = fmt.Sprintf("Delete %s/%s?", r.Repo, r.Branch)
			m.mode = modeConfirmBranchDelete
		}

	case "p":
		// Delete every fully-pushed branch — safe, recoverable from origin.
		plan := m.branchesToDelete(func(r BranchRow) bool { return r.State == BranchPushed })
		if len(plan) == 0 {
			m.notice = "no fully-pushed branches to delete"
			return m, nil
		}
		m.branchPlan = plan
		m.branchPlanMsg = fmt.Sprintf("Delete %d pushed branch(es) across %d repo(s)? Recoverable from origin.",
			len(plan), countBranchRepos(plan))
		m.mode = modeConfirmBranchDelete

	case "D":
		// Delete every branch except main/master (and the checked-out one).
		plan := m.branchesToDelete(func(r BranchRow) bool { return true })
		if len(plan) == 0 {
			m.notice = "nothing to delete"
			return m, nil
		}
		m.branchPlan = plan
		m.branchPlanMsg = fmt.Sprintf("Delete ALL %d non-main branch(es) across %d repo(s)? Includes unpushed work!",
			len(plan), countBranchRepos(plan))
		m.mode = modeConfirmBranchDelete
	}
	return m, nil
}

func (m *Model) handleConfirmBranchDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		plan := m.branchPlan
		m.branchPlan = nil
		if len(plan) == 0 {
			return m, nil
		}
		m.notice = "deleting…"
		m.noticeErr = false
		return m, deleteBranchesCmd(plan)
	default:
		m.mode = modeNormal
		m.branchPlan = nil
		m.notice = "delete cancelled"
		m.noticeErr = false
	}
	return m, nil
}

func (m *Model) currentBranch() (BranchRow, bool) {
	if m.branchCursor >= 0 && m.branchCursor < len(m.branchRows) {
		return m.branchRows[m.branchCursor], true
	}
	return BranchRow{}, false
}

// branchesToDelete returns the non-protected branches matching keep, i.e. the
// deletion plan for a bulk action.
func (m *Model) branchesToDelete(match func(BranchRow) bool) []BranchRow {
	var out []BranchRow
	for _, r := range m.branchRows {
		if r.Protected {
			continue
		}
		if match(r) {
			out = append(out, r)
		}
	}
	return out
}

func countBranchRepos(plan []BranchRow) int {
	seen := map[string]bool{}
	for _, r := range plan {
		seen[r.Path] = true
	}
	return len(seen)
}

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
// Awaiting QA follow-up). It works from the selected work item first — so merging
// works on a Project View / focus issue whose PR isn't separately selectable — and
// falls back to a standalone PR row in the board.
func (m *Model) mergeTarget() (prItem Item, issue, project int, ok bool) {
	if w, wok := m.currentWork(); wok && w.PR != nil {
		return *w.PR, w.Number, w.Project, true
	}
	if it, iok := m.currentItem(); iok && it.Kind == KindPR {
		if w, wok := m.workForPR(it.Number); wok {
			return it, w.Number, w.Project, true
		}
		return it, 0, 0, true
	}
	return Item{}, 0, 0, false
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
		m.startQA = false
		m.startBranchInput.Blur()
		return m, nil
	case "up", "ctrl+p":
		if m.startCloneCursor > 0 {
			m.startCloneCursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.startCloneCursor < m.startCloneMaxCursor() {
			m.startCloneCursor++
		}
		return m, nil
	case "enter":
		// The virtual last row (non-QA only) starts a fresh clone instead of
		// reusing an existing one.
		if !m.startQA && m.startCloneCursor == len(m.startClones) {
			branch := strings.TrimSpace(m.startBranchInput.Value())
			if branch == "" {
				m.notice = "enter a branch name first"
				m.noticeErr = true
				return m, nil
			}
			m.startBranch = branch
			m.startBranchInput.Blur()
			m.newCloneInput.SetValue("")
			m.newCloneInput.Focus()
			m.mode = modeNewClone
			return m, textinput.Blink
		}
		if len(m.startClones) == 0 {
			m.notice = "no local clone of " + m.repo + " found (set clone_base_dirs in config.json)"
			m.noticeErr = true
			m.mode = modeNormal
			m.startQA = false
			return m, nil
		}
		clone := m.startClones[m.startCloneCursor]
		if m.startQA {
			// QA test launch: no branch, no status write — just seed a repro/verify
			// session in the chosen clone. The QA prompt handles branch/main checkout.
			issue := m.startIssue
			m.mode = modeNormal
			m.startQA = false
			m.notice = fmt.Sprintf("testing #%d in %s…", issue, filepath.Base(clone.Path))
			m.noticeErr = false
			return m, launchSessionCmd(clone.Path, m.startPrompt(issue))
		}
		branch := strings.TrimSpace(m.startBranchInput.Value())
		if branch == "" {
			m.notice = "enter a branch name"
			m.noticeErr = true
			return m, nil
		}
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

// startCloneMaxCursor is the highest selectable index in the clone picker. Non-QA
// starts add one virtual row (index == len(startClones)) for "create new clone".
func (m *Model) startCloneMaxCursor() int {
	if m.startQA {
		return len(m.startClones) - 1
	}
	return len(m.startClones) // the extra "create new" row
}

// handleNewCloneKey drives the new-working-dir prompt: it takes a name, prefixes
// it with "fleet-", clones the repo under the first clone_base_dirs entry, then
// runs the normal Start Work flow in the fresh clone.
func (m *Model) handleNewCloneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Back to the clone picker without losing the branch name.
		m.mode = modeStartWork
		m.newCloneInput.Blur()
		m.startBranchInput.SetValue(m.startBranch)
		m.startBranchInput.Focus()
		return m, nil
	case "enter":
		name := sanitizeCloneName(m.newCloneInput.Value())
		if name == "" {
			m.notice = "enter a name for the new working dir"
			m.noticeErr = true
			return m, nil
		}
		base := "~/projects"
		if len(m.config.CloneBaseDirs) > 0 {
			base = m.config.CloneBaseDirs[0]
		}
		dest := filepath.Join(expandHome(base), "fleet-"+name)
		m.mode = modeNormal
		m.newCloneInput.Blur()
		m.notice = fmt.Sprintf("cloning %s into fleet-%s… (this can take a while)", m.repo, name)
		m.noticeErr = false
		return m, cloneAndStartWorkCmd(m.repo, dest, m.startIssue, m.startProject, m.startBranch)
	default:
		var cmd tea.Cmd
		m.newCloneInput, cmd = m.newCloneInput.Update(msg)
		return m, cmd
	}
}

// sanitizeCloneName trims a user-entered working-dir name to a safe folder
// segment: it drops any leading "fleet-" the user typed (we add it), trims
// slashes/spaces, and keeps it to a single path segment.
func sanitizeCloneName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "fleet-")
	s = strings.Trim(s, "/ ")
	if i := strings.IndexAny(s, "/\\"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// startPrompt builds the role-aware seed prompt for a freshly launched Claude
// session, honoring any per-role override in config.StartPrompts.
func (m *Model) startPrompt(issue int) string {
	data := PromptData{Issue: issue}
	if w, ok := m.workByIssue[issue]; ok {
		data.Title, data.URL, data.Branch = w.Title, w.URL, w.Branch
	}
	return renderStartPrompt(m.config.EffectiveRole(), m.config, data)
}

func (m *Model) currentItem() (Item, bool) {
	if m.cursor >= 0 && m.cursor < len(m.flat) {
		return m.flat[m.cursor], true
	}
	return Item{}, false
}
