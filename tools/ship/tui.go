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
		return "⏸ paused"
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
)

// model is the root bubbletea model. It owns the active screen, shared
// state, persisted config, and the engine that drives Fleet.
type model struct {
	width  int
	height int

	screen   screen
	state    runState
	cfg      Config
	priv     string // MDM server private key, in-memory copy
	needWiz  bool
	repoRoot string

	doc  doctorModel
	wiz  wizardModel
	dash dashboardModel

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
		m.state = stateRunning
		m.dash.markRunning(msg.NgrokURL, time.Now())
		return m, m.eng.listen()

	case runtimeFailedMsg:
		m.state = stateError
		m.errMsg = msg.Err.Error()
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
		}
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
	}
	return ""
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
