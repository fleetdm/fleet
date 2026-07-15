package jarvis

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"fleetdm/gm/pkg/ghapi"
)

type viewState int

const (
	stateLoading viewState = iota
	stateLoaded
	stateError
)

// uiMode is the interaction mode within the loaded dashboard.
type uiMode int

const (
	modeNormal       uiMode = iota
	modeSnooze              // picking a snooze duration
	modeConfirmMerge        // confirming a merge
	modeComment             // typing a comment
	modeStartWork           // naming a branch + picking a clone to start work in
)

// Model is the Bubble Tea model for the jarvis dashboard.
type Model struct {
	repo  string
	limit int

	state   viewState
	mode    uiMode
	spinner spinner.Model
	err     error

	login    string
	board    Board // raw, unfiltered
	filtered Board // after triage filtering (what's rendered)
	flat     []Item
	cursor   int
	hidden   int

	triage     *TriageStore
	links      *LinkStore
	focus      *FocusStore
	showHidden bool

	// Issue-centric overlay, rebuilt from the board + stores + cached statuses.
	statuses      map[int]string
	projects      map[int]int
	issueProjects map[int][]ProjectRef // issue number → projects it's on (+ updatedAt)
	localBranches map[string]string    // branch name → local clone folder that has it
	work          []WorkItem
	workByIssue   map[int]WorkItem

	// Focus view: an issue-centric card list of pinned work items.
	focusView   bool
	focusList   []WorkItem
	focusCursor int

	commentInput textinput.Model
	notice       string
	noticeErr    bool

	config    *Config
	cachePath string
	noCache   bool
	fromCache bool // the currently displayed data came from the on-disk cache

	// Start Work modal state.
	startIssue       int
	startProject     int
	startBranchInput textinput.Model
	startClones      []CloneStatus
	startCloneCursor int

	// mergeCherryPick marks the pending merge confirmation as a merge + cherry-pick.
	mergeCherryPick bool

	lastRefresh time.Time
	width       int
	height      int
}

// NewModel constructs a dashboard model for the given repo. When noCache is false
// and a fresh (<CacheTTL) cache exists, the model opens from it without hitting
// GitHub; otherwise it starts in the loading state and fetches live.
func NewModel(repo string, limit int, noCache bool) *Model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "comment…"
	ti.CharLimit = 2000

	bi := textinput.New()
	bi.Placeholder = "branch-name"
	bi.CharLimit = 120

	triage, _ := LoadTriageStore(DefaultTriagePath())
	links, _ := LoadLinkStore(DefaultLinkPath())
	focus, _ := LoadFocusStore(DefaultFocusPath())
	config := LoadConfig(DefaultConfigPath())

	m := &Model{
		repo:             repo,
		limit:            limit,
		state:            stateLoading,
		spinner:          s,
		triage:           triage,
		links:            links,
		focus:            focus,
		config:           config,
		cachePath:        DefaultCachePath(),
		noCache:          noCache,
		commentInput:     ti,
		startBranchInput: bi,
		statuses:         map[int]string{},
		projects:         map[int]int{},
		issueProjects:    map[int][]ProjectRef{},
		width:            100,
		height:           30,
	}

	if !noCache {
		if res, at, ok := LoadCache(m.cachePath); ok && time.Since(at) < CacheTTL {
			m.applyFetch(res)
			m.lastRefresh = at
			m.state = stateLoaded
			m.fromCache = true
		}
	}
	return m
}

// applyFetch loads a fetch result into the model and rebuilds the derived views.
func (m *Model) applyFetch(res FetchResult) {
	m.login = res.Login
	m.board = res.Board
	if res.Statuses != nil {
		m.statuses = res.Statuses
	}
	if res.Projects != nil {
		m.projects = res.Projects
	}
	if res.IssueProjects != nil {
		m.issueProjects = res.IssueProjects
	}
	if res.LocalBranches != nil {
		m.localBranches = res.LocalBranches
	}
	m.autoMarkCompleted()
	m.rebuild()
}

// key returns the triage-store key for an item, scoped by repo and kind.
func (m *Model) key(it Item) string {
	switch it.Kind {
	case KindSession:
		return "session:" + it.SessionID
	case KindPR:
		return fmt.Sprintf("%s/pr:%d", m.repo, it.Number)
	case KindProject:
		return fmt.Sprintf("%s/project:%d", m.repo, it.Number)
	default:
		return fmt.Sprintf("%s/issue:%d", m.repo, it.Number)
	}
}

// rebuild recomputes the filtered board and flat list from the raw board, the
// triage store, and the show-hidden toggle.
func (m *Model) rebuild() {
	now := time.Now()
	filtered := Board{Buckets: map[Bucket][]Item{}}
	hidden := 0
	for _, bk := range BucketOrder {
		for _, it := range m.board.Buckets[bk] {
			vis := m.triage.Visible(m.key(it), it.Updated, now)
			if !vis {
				hidden++
			}
			if vis || m.showHidden {
				filtered.Buckets[bk] = append(filtered.Buckets[bk], it)
			}
		}
	}
	m.filtered = filtered
	m.flat = filtered.Flat()
	m.hidden = hidden
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Rebuild the issue-centric overlay from the raw board + stores + cached
	// statuses (cheap; no network) so pin/unpin and status writes reflect at once.
	m.work = BuildWorkItems(m.board, m.links, m.focus, m.statuses, m.projects)
	m.workByIssue = make(map[int]WorkItem, len(m.work))
	m.focusList = m.focusList[:0]
	for _, w := range m.work {
		m.workByIssue[w.Number] = w
		if w.Focused {
			m.focusList = append(m.focusList, w)
		}
	}
	if m.focusCursor >= len(m.focusList) {
		m.focusCursor = len(m.focusList) - 1
	}
	if m.focusCursor < 0 {
		m.focusCursor = 0
	}
}

// mostRecentProject returns the number of the most recently updated project the
// issue belongs to, or 0 if the issue isn't on any known project. Ties (or
// missing timestamps) break toward the lowest project number for determinism.
func (m *Model) mostRecentProject(issue int) int {
	var best int
	var bestUpdated time.Time
	for _, ref := range m.issueProjects[issue] {
		u := parseTime(ref.UpdatedAt)
		if best == 0 || u.After(bestUpdated) || (u.Equal(bestUpdated) && ref.Number < best) {
			best, bestUpdated = ref.Number, u
		}
	}
	return best
}

// orgProjectURL builds the browser URL for an org project board by its number.
func (m *Model) orgProjectURL(number int) string {
	org := "fleetdm"
	if i := strings.IndexByte(m.repo, '/'); i > 0 {
		org = m.repo[:i]
	}
	return fmt.Sprintf("https://github.com/orgs/%s/projects/%d", org, number)
}

// cherryPickClone picks a working directory for a cherry-pick session: the clone
// already linked to the issue, else a free discovered clone, else any clone, else
// "" (Claude launches in the current directory).
func (m *Model) cherryPickClone(issue int) string {
	if issue != 0 {
		if w, ok := m.workByIssue[issue]; ok && w.ClonePath != "" {
			return w.ClonePath
		}
	}
	clones := DiscoverClones(m.config.CloneBaseDirs, m.repo)
	for _, c := range clones {
		if c.Free() {
			return c.Path
		}
	}
	if len(clones) > 0 {
		return clones[0].Path
	}
	return ""
}

// replacePR swaps a single re-fetched PR into the board: it removes the existing
// PR item (preserving its session linkage) and re-classifies the fresh PR, so a
// small refresh can move it between buckets (e.g. CI went green → Quick Wins).
func (m *Model) replacePR(pr ghapi.PullRequest) {
	var linked Item
	for bk := range m.board.Buckets {
		items := m.board.Buckets[bk]
		for i := range items {
			if items[i].Kind == KindPR && items[i].Number == pr.Number {
				linked = items[i]
				m.board.Buckets[bk] = append(items[:i], items[i+1:]...)
				break
			}
		}
	}
	var it Item
	if strings.EqualFold(pr.Author.Login, m.login) {
		it = classifyMyPR(pr, m.login)
	} else {
		it = classifyReviewRequest(pr, m.login)
	}
	// Carry over the Claude-session linkage computed by the last full fetch.
	if linked.HasSession {
		it.HasSession, it.SessionID, it.Cwd = true, linked.SessionID, linked.Cwd
	}
	m.board.AddItems([]Item{it})
}

// currentFetchResult reconstructs a FetchResult from live model state, so a small
// refresh can keep the on-disk cache consistent.
func (m *Model) currentFetchResult() FetchResult {
	return FetchResult{
		Login: m.login, Board: m.board,
		Statuses: m.statuses, Projects: m.projects, IssueProjects: m.issueProjects,
	}
}

// isCompleted reports whether a board item is finished — a merged/closed PR or a
// closed issue — based on the state we actually have. Cherry-pick items carry no
// PR pointer, so they're intentionally never treated as completed.
func isCompleted(it Item) bool {
	switch it.Kind {
	case KindPR:
		return it.PR != nil && (strings.EqualFold(it.PR.State, "MERGED") || strings.EqualFold(it.PR.State, "CLOSED"))
	case KindIssue:
		return it.Issue != nil && strings.EqualFold(it.Issue.State, "CLOSED")
	}
	return false
}

// autoMarkCompleted marks any merged/closed items done so they clear from the
// board. Called after each fetch/refresh; a no-op for the common open-only fetch,
// but it clears items a per-item refresh reveals as finished.
func (m *Model) autoMarkCompleted() {
	now := time.Now()
	changed := false
	for _, bk := range BucketOrder {
		for _, it := range m.board.Buckets[bk] {
			if isCompleted(it) && m.triage.Visible(m.key(it), it.Updated, now) {
				m.triage.Done(m.key(it), it.Updated)
				changed = true
			}
		}
	}
	if changed {
		_ = m.triage.Save()
	}
}

// branchFolder returns the local clone folder for a work item's branch: the
// recorded clone if Start Work created it, else any clone found to hold the branch.
func (m Model) branchFolder(w WorkItem) string {
	if w.ClonePath != "" {
		return filepath.Base(w.ClonePath)
	}
	if w.Branch != "" {
		return m.localBranches[w.Branch]
	}
	return ""
}

// workForPR returns the work item whose linked PR has the given number.
func (m *Model) workForPR(prNum int) (WorkItem, bool) {
	for _, w := range m.work {
		if w.PR != nil && w.PR.Number == prNum {
			return w, true
		}
	}
	return WorkItem{}, false
}

// currentWork returns the work item the user is acting on: the selected focus
// card in focus view, or the work item behind the highlighted issue in board view.
func (m *Model) currentWork() (WorkItem, bool) {
	if m.focusView {
		if m.focusCursor >= 0 && m.focusCursor < len(m.focusList) {
			return m.focusList[m.focusCursor], true
		}
		return WorkItem{}, false
	}
	if it, ok := m.currentItem(); ok && it.Kind == KindIssue {
		w, ok := m.workByIssue[it.Number]
		return w, ok
	}
	return WorkItem{}, false
}

// fetchDoneMsg carries the result of a background fetch.
type fetchDoneMsg struct {
	res FetchResult
	err error
}

// actionResultMsg carries the result of a GitHub write (merge/comment).
type actionResultMsg struct {
	verb    string
	key     string // triage key to mark done on success (merge)
	issue   int    // linked issue to advance to Awaiting QA on merge (0 if none)
	project int    // board owning that issue's status
	pr      int    // PR number (for the cherry-pick follow-up)
	// cherryPick, when set, launches a cherry-pick Claude session in clonePath
	// after a successful merge instead of the Awaiting QA status bump.
	cherryPick bool
	clonePath  string
	out        string
	err        error
}

// sessionReturnedMsg fires when a resumed Claude session exits.
type sessionReturnedMsg struct{ err error }

// startWorkDoneMsg carries the result of branch creation + status set.
type startWorkDoneMsg struct {
	issue     int
	project   int
	clonePath string
	branch    string
	statusSet string
	warn      string
	err       error
}

// statusWriteMsg carries the result of a standalone status transition.
type statusWriteMsg struct {
	issue     int
	statusSet string
	err       error
}

// itemRefreshedMsg carries fresh data for a single highlighted item (small refresh).
type itemRefreshedMsg struct {
	kind    Kind
	number  int
	pr      *ghapi.PullRequest // KindPR
	status  string             // KindIssue
	project int                // KindIssue
	refs    []ProjectRef       // KindIssue
	closed  bool               // KindIssue: the issue is closed on GitHub
	err     error
}

func (m *Model) fetchCmd() tea.Cmd {
	repo, limit := m.repo, m.limit
	primary := m.config.PrimaryProjects
	baseDirs := m.config.CloneBaseDirs
	return func() tea.Msg {
		res, err := Fetch(repo, limit, primary)
		if err == nil {
			res.LocalBranches = LocalBranchFolders(baseDirs, repo)
		}
		return fetchDoneMsg{res: res, err: err}
	}
}

func mergeCmd(repo string, it Item, key string, issue, project int) tea.Cmd {
	return func() tea.Msg {
		out, err := ghapi.MergePR(repo, it.Number, "squash")
		return actionResultMsg{verb: "merge", key: key, issue: issue, project: project, out: out, err: err}
	}
}

// mergeCherryPickCmd merges the PR, then (on success) the handler launches a
// Claude cherry-pick session in clonePath for the merged PR.
func mergeCherryPickCmd(repo string, it Item, key, clonePath string) tea.Cmd {
	return func() tea.Msg {
		out, err := ghapi.MergePR(repo, it.Number, "squash")
		return actionResultMsg{
			verb: "merge", key: key, pr: it.Number,
			cherryPick: true, clonePath: clonePath, out: out, err: err,
		}
	}
}

// cherryPickPrompt seeds a Claude session to cherry-pick a just-merged PR,
// deferring the target branch choice to the user via the cherry-pick skill.
func cherryPickPrompt(pr int) string {
	return fmt.Sprintf("PR #%d just merged to main. Cherry-pick it into a release branch using the cherry-pick skill. "+
		"First ask me which target/RC branch to cherry-pick into before proceeding.", pr)
}

func commentCmd(repo string, it Item, body string) tea.Cmd {
	return func() tea.Msg {
		var out string
		var err error
		if it.Kind == KindPR {
			out, err = ghapi.CommentPR(repo, it.Number, body)
		} else {
			out, err = ghapi.CommentIssue(repo, it.Number, body)
		}
		return actionResultMsg{verb: "comment", out: out, err: err}
	}
}

// resumeSessionCmd suspends the TUI and resumes a Claude session in its cwd.
func resumeSessionCmd(id, cwd string) tea.Cmd {
	c := exec.Command("claude", "--resume", id)
	if cwd != "" {
		c.Dir = cwd
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return sessionReturnedMsg{err: err}
	})
}

// launchSessionCmd suspends the TUI and starts a fresh Claude session in cwd,
// seeding it with an initial prompt. On exit the dashboard refreshes.
func launchSessionCmd(cwd, prompt string) tea.Cmd {
	var c *exec.Cmd
	if prompt != "" {
		c = exec.Command("claude", prompt)
	} else {
		c = exec.Command("claude")
	}
	c.Dir = cwd
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return sessionReturnedMsg{err: err}
	})
}

// startWorkCmd creates the branch and sets the issue In progress in the background.
func startWorkCmd(issue, project int, clonePath, branch string) tea.Cmd {
	return func() tea.Msg {
		statusSet, warn, err := StartWork(issue, project, clonePath, branch)
		return startWorkDoneMsg{
			issue: issue, project: project, clonePath: clonePath, branch: branch,
			statusSet: statusSet, warn: warn, err: err,
		}
	}
}

// refreshPRCmd re-fetches a single PR's mergeability/CI/review state + threads.
func refreshPRCmd(repo string, number int) tea.Cmd {
	return func() tea.Msg {
		pr, err := ghapi.GetPullRequest(repo, number)
		if err != nil {
			return itemRefreshedMsg{kind: KindPR, number: number, err: err}
		}
		if c, e := ghapi.GetUnresolvedReviewThreadCount(repo, number); e == nil {
			pr.UnresolvedThreads = c
		}
		return itemRefreshedMsg{kind: KindPR, number: number, pr: &pr}
	}
}

// refreshIssueCmd re-fetches a single issue's project Status + board memberships,
// and its open/closed state (so a since-closed issue gets marked done). When
// project is non-zero (e.g. a Project View row), the status is read from THAT
// project so it never shows a secondary project's column; otherwise the workflow
// board is picked heuristically.
func refreshIssueCmd(repo string, number, project int) tea.Cmd {
	return func() tea.Msg {
		found, err := ghapi.GetAllIssueProjectStatuses(number)
		if err != nil {
			return itemRefreshedMsg{kind: KindIssue, number: number, err: err}
		}
		pid, status := project, ""
		if ps, ok := found[project]; project != 0 && ok && ps.Present {
			status = ps.Status
		} else {
			pid, status = pickWorkflowStatus(found)
		}
		var refs []ProjectRef
		for p, ps := range found {
			if ps.Present {
				refs = append(refs, ProjectRef{Number: p, UpdatedAt: ps.UpdatedAt, Title: ps.Title})
			}
		}
		closed := false
		if iss, e := ghapi.GetIssue(repo, number); e == nil {
			closed = strings.EqualFold(iss.State, "CLOSED")
		}
		return itemRefreshedMsg{kind: KindIssue, number: number, status: status, project: pid, refs: refs, closed: closed}
	}
}

// setStatusCmd performs a standalone status transition (e.g. Ready for review,
// Awaiting QA), trying each candidate intent against the board's Status options.
func setStatusCmd(issue, project int, intents []string) tea.Cmd {
	return func() tea.Msg {
		statusSet, err := resolveAndSetStatus(issue, project, intents)
		return statusWriteMsg{issue: issue, statusSet: statusSet, err: err}
	}
}

// openURLCmd opens a work item in the browser. Read-only.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return nil
		}
		opener := "xdg-open"
		if runtime.GOOS == "darwin" {
			opener = "open"
		}
		_ = exec.Command(opener, url).Start()
		return nil
	}
}

func (m Model) Init() tea.Cmd {
	if m.state == stateLoaded {
		return nil // opened from a fresh cache; no network until r
	}
	return tea.Batch(m.spinner.Tick, m.fetchCmd())
}

// Run launches the dashboard TUI. When noCache is true it ignores any cached
// fetch and pulls live on startup.
func Run(repo string, limit int, noCache bool) error {
	m := NewModel(repo, limit, noCache)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
