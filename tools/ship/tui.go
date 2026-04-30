package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// runState is the headline status shown in the TUI header.
type runState int

const (
	stateIdle     runState = iota // not yet started
	stateRunning                  // fleet serving normally
	stateBuilding                 // rebuild + restart in progress
	statePaused                   // auto-restart paused for a demo
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

// model is the root bubbletea model. It owns the active screen, shared state,
// and the persisted config. Sub-screens are dispatched to via the screen field.
type model struct {
	width    int
	height   int
	screen   screen
	state    runState
	cfg      Config
	needWiz  bool
	doc      doctorModel
	wiz      wizardModel
	dash     dashboardModel
	err      string // surfaced if config save fails after the wizard
	quitting bool
}

func initialModel(cfg Config, hasPrivateKey, needWiz bool) model {
	return model{
		screen:  screenDoctor,
		state:   stateIdle,
		cfg:     cfg,
		needWiz: needWiz,
		doc:     newDoctorModel(context.Background()),
		wiz:     newWizardModel(cfg, hasPrivateKey),
		dash:    newDashboardModel(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global shortcuts that apply on every screen.
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			// Don't quit on bare "q" while typing into a text field — the
			// wizard's text inputs need that key. The dashboard and doctor
			// can take it.
			if m.screen != screenWizard {
				m.quitting = true
				return m, tea.Quit
			}
		}

		// Per-screen handling.
		switch m.screen {
		case screenDoctor:
			if msg.String() == "enter" {
				if m.needWiz {
					m.screen = screenWizard
				} else {
					m.screen = screenDashboard
				}
			}
			return m, nil

		case screenWizard:
			updated, cmd := m.wiz.update(msg)
			m.wiz = updated
			if m.wiz.done {
				if err := m.persistWizard(); err != nil {
					m.err = err.Error()
				}
				m.screen = screenDashboard
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

// persistWizard writes the wizard's results to disk. Called once when the
// wizard signals done.
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
		return m.dash.view(m.width, m.state)
	}
	return ""
}
