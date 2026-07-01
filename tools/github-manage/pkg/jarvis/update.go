package jarvis

import (
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
		m.login = msg.res.Login
		m.board = msg.res.Board
		m.lastRefresh = time.Now()
		m.state = stateLoaded
		m.rebuild()
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
		}
		return m, nil

	case sessionReturnedMsg:
		// Returned from a resumed Claude session; refresh in case state changed.
		m.state = stateLoading
		return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

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
	default:
		return m.handleNormalKey(msg)
	}
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = "" // any keypress clears a stale notice
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "r":
		m.state = stateLoading
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.flat)-1 {
			m.cursor++
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.flat) > 0 {
			m.cursor = len(m.flat) - 1
		}

	case "H":
		m.showHidden = !m.showHidden
		m.rebuild()

	case "enter", "o":
		if it, ok := m.currentItem(); ok {
			if it.Kind == KindSession {
				return m, resumeSessionCmd(it.SessionID, it.Cwd)
			}
			return m, openURLCmd(it.URL)
		}

	case "J":
		// Jump into a linked Claude session (on a PR/issue) or a session item.
		if it, ok := m.currentItem(); ok && (it.HasSession || it.Kind == KindSession) {
			return m, resumeSessionCmd(it.SessionID, it.Cwd)
		}

	case "s":
		if _, ok := m.currentItem(); ok {
			m.mode = modeSnooze
		}
	case "d":
		if it, ok := m.currentItem(); ok {
			m.triage.Dismiss(m.key(it), it.Updated)
			_ = m.triage.Save()
			m.notice = "dismissed"
			m.rebuild()
		}
	case "x":
		if it, ok := m.currentItem(); ok {
			m.triage.Done(m.key(it), it.Updated)
			_ = m.triage.Save()
			m.notice = "marked done"
			m.rebuild()
		}
	case "u":
		if it, ok := m.currentItem(); ok {
			m.triage.Clear(m.key(it))
			_ = m.triage.Save()
			m.notice = "cleared"
			m.rebuild()
		}

	case "m":
		if it, ok := m.currentItem(); ok && it.Kind == KindPR {
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
		if it, ok := m.currentItem(); ok && it.Kind == KindPR {
			m.notice = "merging…"
			return m, mergeCmd(m.repo, it, m.key(it))
		}
	default:
		m.mode = modeNormal
	}
	return m, nil
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

func (m *Model) currentItem() (Item, bool) {
	if m.cursor >= 0 && m.cursor < len(m.flat) {
		return m.flat[m.cursor], true
	}
	return Item{}, false
}
