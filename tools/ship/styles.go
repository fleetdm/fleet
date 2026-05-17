package main

import "github.com/charmbracelet/lipgloss"

// Brand palette pulled from frontend/styles/var/colors.scss. Every role uses
// AdaptiveColor so the TUI reads well on both light- and dark-background
// terminals — values map to the corresponding light- or dark-mode CSS token.
//
// Accent is core-fleet-green (Fleet's 2025 brand color, replacing the older
// vibrant-blue). Status colors use ui-success / ui-warning / ui-error, which
// are intentionally distinct shades from the brand green so "running" status
// dots don't blur into accent text.
var (
	// Outer pane / divider lines. Want it visible but not loud on either
	// terminal background.
	colorBorder = lipgloss.AdaptiveColor{Light: "#515774", Dark: "#6a6f8a"} // ui-fleet-black-75 / dark core-fleet-blue

	// Labels, helper text, "(waiting...)" placeholders.
	colorMuted = lipgloss.AdaptiveColor{Light: "#8b8fa2", Dark: "#87888B"} // ui-fleet-black-50

	// Brand accent: header text, URLs, keybind letters.
	colorAccent = lipgloss.AdaptiveColor{Light: "#009a7d", Dark: "#00C28B"} // core-fleet-green

	// Status colors.
	colorOK   = lipgloss.AdaptiveColor{Light: "#3db67b", Dark: "#4dc98b"} // ui-success
	colorWarn = lipgloss.AdaptiveColor{Light: "#ebbc43", Dark: "#f0ca5e"} // ui-warning
	colorErr  = lipgloss.AdaptiveColor{Light: "#d66c7b", Dark: "#e07888"} // ui-error
)

var (
	// Outer pane border that wraps the whole TUI.
	stylePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Header bar inside the pane: app name on the left, state on the right.
	styleHeaderBrand = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleHeaderState = lipgloss.NewStyle().Bold(true)

	// Status block: "label  value" rows.
	styleLabel = lipgloss.NewStyle().Foreground(colorMuted)
	styleValue = lipgloss.NewStyle()
	styleURL   = lipgloss.NewStyle().Foreground(colorAccent).Underline(true)

	// Section divider line (e.g. above "fleet server" pane).
	styleSection = lipgloss.NewStyle().Foreground(colorMuted)

	// Keybinding hint row.
	styleKey  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleHint = lipgloss.NewStyle().Foreground(colorMuted)
)

// stateStyle returns the right header color for the current run state.
func stateStyle(s runState) lipgloss.Style {
	switch s {
	case stateRunning:
		return styleHeaderState.Foreground(colorOK)
	case stateBuilding, statePaused:
		return styleHeaderState.Foreground(colorWarn)
	case stateError:
		return styleHeaderState.Foreground(colorErr)
	default:
		return styleHeaderState.Foreground(colorMuted)
	}
}
