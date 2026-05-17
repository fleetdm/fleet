package main

import "strings"

// logSource identifies which captured log buffer the overlay shows.
type logSource int

const (
	logSourceFleet logSource = iota
	logSourceWebpack
)

func (s logSource) title() string {
	switch s {
	case logSourceFleet:
		return "fleet server logs"
	case logSourceWebpack:
		return "webpack logs"
	}
	return "logs"
}

// renderLogScreen takes a buffered slice of lines and renders a
// full-width overlay. We trim to whatever fits the height so the
// keybind row at the bottom stays visible.
func renderLogScreen(width, height int, source logSource, lines []string) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	header := styleHeaderBrand.Render("Fleet ship") +
		styleHint.Render("  ·  "+source.title())
	hints := styleKey.Render("esc") + " " + styleHint.Render("back") +
		styleHint.Render("   ·   ") +
		styleKey.Render("q") + " " + styleHint.Render("quit")

	// Reserve four lines: header, blank, blank, hints. Whatever's left is
	// for log content.
	content := lines
	available := height - 6 // 4 framing rows + 2 pane border lines
	if available < 5 {
		available = 5
	}
	if len(content) > available {
		content = content[len(content)-available:]
	}
	if len(content) == 0 {
		content = []string{styleHint.Render("(no output yet)")}
	}

	body := []string{
		header,
		"",
		strings.Join(content, "\n"),
		"",
		hints,
	}
	return stylePane.Width(width - 2).Render(strings.Join(body, "\n"))
}

