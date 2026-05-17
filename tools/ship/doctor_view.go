package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// doctorModel renders the preflight check screen.
type doctorModel struct {
	checks []Check
}

func newDoctorModel(ctx context.Context) doctorModel {
	return doctorModel{checks: runChecks(ctx)}
}

// allOK is a convenience used by the root model to decide whether to enable
// the "continue" path on enter.
func (d doctorModel) allOK() bool { return AllOK(d.checks) }

func (d doctorModel) view(width int) string {
	if width <= 0 {
		width = 80
	}

	header := styleHeaderBrand.Render("Fleet ship") + styleHint.Render("  ·  checking your machine")

	rows := make([]string, 0, len(d.checks))
	for _, c := range d.checks {
		rows = append(rows, renderCheckRow(c))
	}

	body := []string{header, "", strings.Join(rows, "\n")}

	if !d.allOK() {
		body = append(body,
			"",
			styleHint.Render("How to fix: see ")+styleURL.Render("tools/ship/README.md#prerequisites"),
			"",
			styleKey.Render("enter")+" "+styleHint.Render("continue anyway")+styleHint.Render("   ·   ")+
				styleKey.Render("q")+" "+styleHint.Render("quit"),
		)
	} else {
		body = append(body,
			"",
			styleOK().Render("All set."),
			"",
			styleKey.Render("enter")+" "+styleHint.Render("continue")+styleHint.Render("   ·   ")+
				styleKey.Render("q")+" "+styleHint.Render("quit"),
		)
	}

	return stylePane.Width(width - 2).Render(strings.Join(body, "\n"))
}

// renderCheckRow lays out one check as: " ICON  Name (padded)  Detail "
func renderCheckRow(c Check) string {
	icon := lipgloss.NewStyle().Foreground(colorOK).Bold(true).Render("✓")
	switch c.Status {
	case CheckMissing:
		icon = lipgloss.NewStyle().Foreground(colorErr).Bold(true).Render("✗")
	case CheckWarn:
		icon = lipgloss.NewStyle().Foreground(colorWarn).Bold(true).Render("⚠")
	}
	nameCol := lipgloss.NewStyle().Width(20).Render(c.Name)
	detail := styleHint.Render(c.Detail)
	return fmt.Sprintf("  %s  %s%s", icon, nameCol, detail)
}

// styleOK is a small helper for one-off "all good" messages — separate from
// the row-icon styling so we don't have to thread a width into it.
func styleOK() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorOK).Bold(true)
}
