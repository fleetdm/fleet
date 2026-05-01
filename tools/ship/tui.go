package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// runState is the headline status shown in the TUI header.
type runState int

const (
	stateIdle     runState = iota // not yet started
	stateRunning                  // fleet serving normally
	stateBuilding                 // start sequence (or rebuild) in progress
	statePaused                   // auto-restart paused for a demo (PR 2+)
	stateError                    // last operation failed
)

func (s runState) String() string {
	switch s {
	case stateRunning:
		return "● running"
	case stateBuilding:
		return "◐ building"
	case statePaused:
		return "⏸ paused rebuilds"
	case stateError:
		return "✗ error"
	default:
		return "· idle"
	}
}

// screen identifies which screen is currently being rendered.
type screen int

const (
	screenDoctor screen = iota
	screenWizard
	screenDashboard
	screenLogs        // overlay shown via l/w; esc returns to dashboard
	screenSwitcher    // worktree list, opened with `s`
	screenNewWorktree // form to create a new worktree, opened from switcher with `n`
)

// model is the root bubbletea model. It owns the active screen, shared
// state, persisted config, and the engine that drives Fleet.
type model struct {
	width  int
	height int

	screen   screen
	state    runState
	cfg      Config
	priv     string // Fleet server private key, in-memory copy
	needWiz  bool
	repoRoot string

	doc  doctorModel
	wiz  wizardModel
	dash dashboardModel
	sw   switcherModel
	nw   newWorktreeModel

	// logSource selects which buffer the screenLogs overlay shows.
	logSource logSource

	eng *engine

	// errMsg surfaces fatal errors (config save, startup, etc.) so the user
	// sees something instead of a hung screen.
	errMsg   string
	quitting bool
}

func initialModel(cfg Config, privateKey string, hasPrivateKey, needWiz bool, repoRoot string) model {
	return model{
		screen:   screenDoctor,
		state:    stateIdle,
		cfg:      cfg,
		priv:     privateKey,
		needWiz:  needWiz,
		repoRoot: repoRoot,
		doc:      newDoctorModel(context.Background()),
		wiz:      newWizardModel(cfg, hasPrivateKey),
		dash:     newDashboardModel(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case stepUpdateMsg:
		m.dash.applyStepUpdate(msg)
		switch msg.Status {
		case stepRunning, stepPending:
			m.state = stateBuilding
		case stepFailed:
			m.state = stateError
		}
		return m, m.eng.listen()

	case logLineMsg:
		m.dash.appendLog(logLine(msg))
		return m, m.eng.listen()

	case runtimeReadyMsg:
		// Preserve paused state if the user was paused before a manual
		// restart finished — they shouldn't get auto-unpaused by it.
		if m.state != statePaused {
			m.state = stateRunning
		}
		m.dash.markRunning(msg.NgrokURL, time.Now())
		return m, m.eng.listen()

	case runtimeFailedMsg:
		m.state = stateError
		m.errMsg = msg.Err.Error()
		return m, m.eng.listen()

	case rebuildStartedMsg:
		m.state = stateBuilding
		m.errMsg = ""
		m.dash.beginRebuild(msg.Reason)
		return m, m.eng.listen()

	case switchStartedMsg:
		m.state = stateBuilding
		m.errMsg = ""
		m.dash.beginSwitch(msg.Reason)
		// Ensure we're showing the dashboard so the user sees progress.
		m.screen = screenDashboard
		// Update the active worktree displayed in the status block right
		// away — the engine has already swapped its repoRoot.
		newPath := findPathByName(m.cfg.Worktrees, msg.ToName)
		if newPath != "" {
			m.repoRoot = newPath
			m.dash.setIdentity(msg.ToName, currentBranch(newPath), "fleet")
		}
		// Persist the new active worktree.
		m.cfg.ActiveWorktree = msg.ToName
		_ = SaveConfig(m.cfg)
		return m, m.eng.listen()

	case pauseChangedMsg:
		if msg.Paused {
			m.state = statePaused
		} else if m.state == statePaused {
			// becoming unpaused; if a rebuild is about to start the
			// next message will flip to stateBuilding, otherwise we're
			// back to running.
			m.state = stateRunning
		}
		m.dash.setQueued(msg.Queued)
		return m, m.eng.listen()

	case tea.KeyMsg:
		// Global shortcuts.
		switch msg.String() {
		case "ctrl+c":
			return m.quit()
		case "q":
			if m.screen != screenWizard {
				return m.quit()
			}
		}

		// Per-screen handling.
		switch m.screen {
		case screenDoctor:
			if msg.String() == "enter" {
				if m.needWiz {
					m.screen = screenWizard
					return m, nil
				}
				m.screen = screenDashboard
				return m.startEngine()
			}
			return m, nil

		case screenWizard:
			updated, cmd := m.wiz.update(msg)
			m.wiz = updated
			if m.wiz.done {
				if err := m.persistWizard(); err != nil {
					m.errMsg = err.Error()
				}
				m.screen = screenDashboard
				m2, startCmd := m.startEngine()
				return m2, tea.Batch(cmd, startCmd)
			}
			return m, cmd

		case screenDashboard:
			// Most dashboard keys only do anything once Fleet is up
			// (running or paused). r is additionally blocked while
			// paused — pressing r during a demo shouldn't trigger a
			// restart.
			if m.state != stateRunning && m.state != statePaused {
				return m, nil
			}
			switch msg.String() {
			case "l":
				m.logSource = logSourceFleet
				m.screen = screenLogs
			case "w":
				m.logSource = logSourceWebpack
				m.screen = screenLogs
			case "n":
				return m, openNgrokInspector()
			case "r":
				if m.state == stateRunning && m.eng != nil {
					m.eng.HandleTrigger("r pressed", nil)
				}
			case "p":
				if m.eng != nil {
					m.eng.TogglePause()
				}
			case "s":
				m.sw = newSwitcherModel(m.cfg.Worktrees, m.cfg.ActiveWorktree)
				m.screen = screenSwitcher
			}
			return m, nil

		case screenLogs:
			if msg.String() == "esc" {
				m.screen = screenDashboard
			}
			return m, nil

		case screenSwitcher:
			next, action := m.sw.onKey(msg.String())
			m.sw = next
			switch action {
			case switcherBack:
				m.screen = screenDashboard
			case switcherSwitch:
				picked := m.sw.entries[m.sw.cursor]
				m.screen = screenDashboard
				if m.eng != nil {
					m.eng.SwitchTo(picked.Path, picked.Name)
				}
			case switcherOpenNew:
				m.nw = newNewWorktreeModel(m.repoRoot)
				m.screen = screenNewWorktree
			case switcherDeleteConfirm:
				picked := m.sw.entries[m.sw.cursor]
				if _, err := gitWorktreeRemove(context.Background(), m.repoRoot, picked.Path, false); err != nil {
					m.errMsg = err.Error()
				} else {
					RemoveWorktree(&m.cfg, picked.Name)
					_ = SaveConfig(m.cfg)
					m.sw = newSwitcherModel(m.cfg.Worktrees, m.cfg.ActiveWorktree)
				}
			}
			return m, nil

		case screenNewWorktree:
			if msg.String() == "esc" {
				m.screen = screenSwitcher
				return m, nil
			}
			next, cmd := m.nw.update(msg)
			m.nw = next
			if m.nw.done {
				UpsertWorktree(&m.cfg, m.nw.created)
				_ = SaveConfig(m.cfg)
				m.sw = newSwitcherModel(m.cfg.Worktrees, m.cfg.ActiveWorktree)
				m.screen = screenSwitcher
			}
			return m, cmd
		}
	}

	// textinput tick messages (cursor blink) flow through the active
	// form-style screens.
	if m.screen == screenNewWorktree {
		next, cmd := m.nw.update(msg)
		m.nw = next
		return m, cmd
	}

	// Pass other messages (e.g. textinput's blink ticks) through to the
	// active screen so its sub-components can update.
	if m.screen == screenWizard {
		updated, cmd := m.wiz.update(msg)
		m.wiz = updated
		return m, cmd
	}
	return m, nil
}

// quit just hands control back to bubbletea — the actual engine teardown
// happens in main() after p.Run() returns, so the user sees the TUI exit
// cleanly instead of a frozen screen during docker-compose-down.
func (m model) quit() (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}

// startEngine spins up the orchestrator and returns a cmd that begins
// pumping its messages into the TUI.
func (m model) startEngine() (model, tea.Cmd) {
	if m.eng != nil {
		return m, nil
	}
	// Populate the dashboard's identifying fields up front — they're
	// known immediately and don't depend on the engine succeeding.
	m.dash.setIdentity(
		filepath.Base(m.repoRoot),
		currentBranch(m.repoRoot),
		"fleet", // default Fleet dev database name
	)

	m.eng = newEngine(runtimeOpts{
		cfg:        m.cfg,
		privateKey: m.priv,
		repoRoot:   m.repoRoot,
	})
	m.dash.beginStart()
	m.state = stateBuilding
	m.eng.Start(context.Background())
	return m, m.eng.listen()
}

// currentBranch reads the current branch from the worktree at root. Returns
// "HEAD" on detached checkouts, empty string if git fails entirely.
func currentBranch(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// persistWizard writes the wizard's results to disk.
func (m *model) persistWizard() error {
	cfg, mdm := m.wiz.applyTo(m.cfg)
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	m.cfg = cfg
	if mdm != "" {
		if err := SavePrivateKey(mdm); err != nil {
			return fmt.Errorf("save private key: %w", err)
		}
		m.priv = mdm
	}
	return nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenDoctor:
		return m.doc.view(m.width)
	case screenWizard:
		return m.wiz.view(m.width)
	case screenDashboard:
		return m.dash.view(m.width, m.state, m.errMsg)
	case screenLogs:
		var lines []string
		switch m.logSource {
		case logSourceFleet:
			lines = m.dash.fleetLog
		case logSourceWebpack:
			lines = m.dash.webpackLog
		}
		return renderLogScreen(m.width, m.height, m.logSource, lines)
	case screenSwitcher:
		return m.sw.view(m.width)
	case screenNewWorktree:
		return m.nw.view(m.width)
	}
	return ""
}

// findPathByName looks up a worktree path in cfg.Worktrees. Returns ""
// if not found.
func findPathByName(entries []WorktreeEntry, name string) string {
	for _, e := range entries {
		if e.Name == name {
			return e.Path
		}
	}
	return ""
}

// openNgrokInspector launches the system browser pointed at ngrok's
// local traffic UI. Wrapped in a tea.Cmd so we don't block Update.
func openNgrokInspector() tea.Cmd {
	return func() tea.Msg {
		_ = exec.Command("open", "http://localhost:4040").Start()
		return nil
	}
}

// findRepoRoot walks up from the current working directory looking for the
// Fleet repo's top-level Makefile + tools/ pair, falling back to git.
func findRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			return root, nil
		}
	}
	// Fallback: assume CWD is tools/ship/, root is two up.
	abs, err := filepath.Abs("../..")
	if err != nil {
		return "", errors.New("could not resolve Fleet repo root")
	}
	return abs, nil
}
