// Package traymenu builds the (pure, testable) model of the macOS tray menu:
// labels, enabled states, and the start/stop toggle. The Wails-specific tray
// wiring renders these Items and routes clicks. Ported from tray.rs.
package traymenu

// State mirrors what the frontend pushes via update_tray.
type State struct {
	Branch        *string `json:"branch"`
	ServeUp       bool    `json:"serve_up"`
	DockerUp      bool    `json:"docker_up"`
	NgrokRunning  bool    `json:"ngrok_running"`
	PythonRunning bool    `json:"python_running"`
}

// Item is one rendered tray menu entry.
type Item struct {
	ID        string
	Label     string
	Enabled   bool
	Separator bool
}

// Native macOS menus can't color-style text, so we lean on emoji for the
// "this is live" signal — closest to the pulsing dot in the main UI.
func dot(on bool) string {
	if on {
		return "🟢"
	}
	return "⚪"
}

// svcLabel is "dot  name" with an optional trailing extra (e.g. a port). The
// dot already encodes up/down, so we don't repeat it in words.
func svcLabel(on bool, name, extra string) string {
	if extra != "" {
		return dot(on) + "  " + name + "  ·  " + extra
	}
	return dot(on) + "  " + name
}

// BranchLabel is the top informational row.
func BranchLabel(s State) string {
	if s.Branch != nil {
		return "Branch: " + *s.Branch
	}
	return "No repo configured"
}

// AnyRunning reports whether any tracked service is up.
func AnyRunning(s State) bool {
	return s.ServeUp || s.DockerUp || s.NgrokRunning || s.PythonRunning
}

// BuildItems returns the full ordered menu for the given state.
func BuildItems(s State) []Item {
	serveExtra := ""
	if s.ServeUp {
		// Keep the port when up (real info the dot doesn't convey); drop it
		// when down.
		serveExtra = ":8080"
	}

	items := []Item{
		{ID: "tray:branch", Label: BranchLabel(s)},
		{Separator: true},
		{ID: "tray:svc-serve", Label: svcLabel(s.ServeUp, "fleet serve", serveExtra)},
		{ID: "tray:svc-docker", Label: svcLabel(s.DockerUp, "docker", "")},
		{ID: "tray:svc-ngrok", Label: svcLabel(s.NgrokRunning, "ngrok", "")},
		{ID: "tray:svc-python", Label: svcLabel(s.PythonRunning, "python", "")},
		{Separator: true},
	}

	if AnyRunning(s) {
		items = append(items, Item{ID: "tray:stop-all", Label: "■ Stop all", Enabled: true})
	} else {
		// Start-all is only actionable once a repo is configured.
		items = append(items, Item{ID: "tray:start-all", Label: "▶ Start all", Enabled: s.Branch != nil})
	}
	items = append(items,
		Item{ID: "tray:show", Label: "Open Fleet Dev", Enabled: true},
		Item{ID: "tray:quit", Label: "Quit", Enabled: true},
	)
	return items
}
