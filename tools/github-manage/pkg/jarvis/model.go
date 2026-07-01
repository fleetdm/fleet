package jarvis

import (
	"fmt"
	"os/exec"
	"runtime"
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
	modeNormal uiMode = iota
	modeSnooze        // picking a snooze duration
	modeConfirmMerge  // confirming a merge
	modeComment       // typing a comment
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
	showHidden bool

	commentInput textinput.Model
	notice       string
	noticeErr    bool

	lastRefresh time.Time
	width       int
	height      int
}

// NewModel constructs a dashboard model for the given repo.
func NewModel(repo string, limit int) *Model {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "comment…"
	ti.CharLimit = 2000

	triage, _ := LoadTriageStore(DefaultTriagePath())

	return &Model{
		repo:         repo,
		limit:        limit,
		state:        stateLoading,
		spinner:      s,
		commentInput: ti,
		triage:       triage,
		width:        100,
		height:       30,
	}
}

// key returns the triage-store key for an item, scoped by repo and kind.
func (m *Model) key(it Item) string {
	switch it.Kind {
	case KindSession:
		return "session:" + it.SessionID
	case KindPR:
		return fmt.Sprintf("%s/pr:%d", m.repo, it.Number)
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
}

// fetchDoneMsg carries the result of a background fetch.
type fetchDoneMsg struct {
	res FetchResult
	err error
}

// actionResultMsg carries the result of a GitHub write (merge/comment).
type actionResultMsg struct {
	verb string
	key  string // triage key to mark done on success (merge)
	out  string
	err  error
}

// sessionReturnedMsg fires when a resumed Claude session exits.
type sessionReturnedMsg struct{ err error }

func (m *Model) fetchCmd() tea.Cmd {
	repo, limit := m.repo, m.limit
	return func() tea.Msg {
		res, err := Fetch(repo, limit)
		return fetchDoneMsg{res: res, err: err}
	}
}

func mergeCmd(repo string, it Item, key string) tea.Cmd {
	return func() tea.Msg {
		out, err := ghapi.MergePR(repo, it.Number, "squash")
		return actionResultMsg{verb: "merge", key: key, out: out, err: err}
	}
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
	return tea.Batch(m.spinner.Tick, m.fetchCmd())
}

// Run launches the dashboard TUI.
func Run(repo string, limit int) error {
	m := NewModel(repo, limit)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
