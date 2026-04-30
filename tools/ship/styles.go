package main

import "github.com/charmbracelet/lipgloss"

// Brand palette pulled from frontend/styles/var/colors.scss.
// Borders and muted text use AdaptiveColor so the TUI reads well on
// both light and dark terminals; the brand accents work on both as-is.
var (
	colorBorder = lipgloss.AdaptiveColor{Light: "#192147", Dark: "#6a6f8a"} // core-fleet-black / dark core-fleet-blue
	colorMuted  = lipgloss.AdaptiveColor{Light: "#515774", Dark: "#8b8fa2"} // ui-fleet-black-75 / ui-fleet-black-50
	colorAccent = lipgloss.Color("#6a67fe")                                 // core-vibrant-blue
	colorOK     = lipgloss.Color("#3db67b")                                 // ui-success
	colorWarn   = lipgloss.Color("#ebbc43")                                 // ui-warning
	colorErr    = lipgloss.Color("#d66c7b")                                 // ui-error
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
