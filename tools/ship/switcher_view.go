package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// switcherModel renders the worktree switcher: a list of registered
// worktrees with the active one marked, and footer hints for new /
// delete / switch / back. Inline delete confirmation lives here too —
// no separate "are you sure?" screen.
type switcherModel struct {
	entries    []WorktreeEntry
	activeName string
	cursor     int

	// confirmingDelete is the cursor index whose deletion we just
	// asked for. -1 when not confirming.
	confirmingDelete int
}

func newSwitcherModel(entries []WorktreeEntry, activeName string) switcherModel {
	cursor := 0
	for i, e := range entries {
		if e.Name == activeName {
			cursor = i
			break
		}
	}
	return switcherModel{
		entries:          entries,
		activeName:       activeName,
		cursor:           cursor,
		confirmingDelete: -1,
	}
}

// switcherAction tells the root model what the switcher decided.
type switcherAction int

const (
	switcherNone switcherAction = iota
	switcherBack
	switcherSwitch        // m.entries[cursor]
	switcherOpenNew       // open new-worktree form
	switcherDeleteConfirm // m.entries[cursor]
)

// onKey returns the next state plus any action the root model should
// take. We don't depend on tea.KeyMsg's import here — the caller
// passes the string form so this file can stay framework-light.
func (s switcherModel) onKey(key string) (switcherModel, switcherAction) {
	if s.confirmingDelete >= 0 {
		switch key {
		case "y", "Y":
			idx := s.confirmingDelete
			s.confirmingDelete = -1
			s.cursor = idx
			return s, switcherDeleteConfirm
		case "n", "N", "esc":
			s.confirmingDelete = -1
			return s, switcherNone
		}
		return s, switcherNone
	}

	switch key {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.entries)-1 {
			s.cursor++
		}
	case "enter":
		if len(s.entries) == 0 {
			return s, switcherOpenNew
		}
		// Don't bother re-switching to the worktree that's already active
		if s.entries[s.cursor].Name == s.activeName {
			return s, switcherBack
		}
		return s, switcherSwitch
	case "n":
		return s, switcherOpenNew
	case "d":
		// Refuse to delete the active worktree — switching away would
		// be the prerequisite, and we don't want to entangle that here.
		if len(s.entries) > 0 && s.entries[s.cursor].Name != s.activeName {
			s.confirmingDelete = s.cursor
		}
	case "esc":
		return s, switcherBack
	}
	return s, switcherNone
}

func (s switcherModel) view(width int) string {
	if width <= 0 {
		width = 80
	}
	header := styleHeaderBrand.Render("Fleet ship") +
		styleHint.Render("  ·  switch worktree")

	rows := make([]string, 0, len(s.entries)+1)
	if len(s.entries) == 0 {
		rows = append(rows, styleHint.Render("  No worktrees yet — press n to create one."))
	}
	for i, e := range s.entries {
		rows = append(rows, renderWorktreeRow(e, i == s.cursor, e.Name == s.activeName))
	}
	rows = append(rows, "")
	rows = append(rows, "  "+styleKey.Render("+")+" "+styleHint.Render("press n to create a new worktree"))

	var hints string
	if s.confirmingDelete >= 0 {
		name := s.entries[s.confirmingDelete].Name
		warn := lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
		rows = append(rows, "")
		rows = append(rows,
			"  "+warn.Render("Delete worktree ")+
				styleValue.Render(name)+
				warn.Render("? This runs `git worktree remove`."))
		hints = styleKey.Render("y") + " " + styleHint.Render("delete") +
			styleHint.Render("   ·   ") +
			styleKey.Render("n") + " " + styleHint.Render("cancel")
	} else {
		hints = strings.Join([]string{
			styleKey.Render("↑↓") + " " + styleHint.Render("select"),
			styleKey.Render("enter") + " " + styleHint.Render("switch"),
			styleKey.Render("n") + " " + styleHint.Render("new"),
			styleKey.Render("d") + " " + styleHint.Render("delete"),
			styleKey.Render("esc") + " " + styleHint.Render("back"),
		}, styleHint.Render(" · "))
	}

	body := strings.Join([]string{
		header,
		"",
		strings.Join(rows, "\n"),
		"",
		hints,
	}, "\n")
	return stylePane.Width(width - 2).Render(body)
}

func renderWorktreeRow(e WorktreeEntry, focused, active bool) string {
	dot := styleHint.Render("  ")
	if active {
		dot = lipgloss.NewStyle().Foreground(colorOK).Bold(true).Render(" ●")
	}
	name := lipgloss.NewStyle().Width(20).Render(e.Name)
	branch := styleHint.Render(e.Branch)
	if e.Branch == "" {
		branch = styleHint.Render("(unknown branch)")
	}
	prefix := "  "
	if focused {
		prefix = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("▸ ")
	}
	return prefix + dot + " " + name + branch
}
