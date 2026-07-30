package main

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/fleetdm/fleet/tools/hangar/internal/traymenu"
)

const mainWindowName = "main"

// trayController owns the native system tray and rebuilds its menu from
// pushed state. Ported from tray.rs.
type trayController struct {
	app  *application.App
	tray *application.SystemTray
}

func newTrayController(app *application.App, icon []byte) *trayController {
	t := app.SystemTray.New()
	// Template mode renders the icon as a system-colored silhouette in the
	// menu bar (alpha-only), matching the rest of the macOS menu bar.
	t.SetTemplateIcon(icon)
	t.SetTooltip("Fleet Hangar")
	c := &trayController{app: app, tray: t}
	c.update(traymenu.State{})
	// Left-click opens the main window; the menu shows on right-click.
	t.OnClick(func() { showMainWindow(app) })
	return c
}

// update rebuilds the tray menu in place from the given state.
func (c *trayController) update(s traymenu.State) {
	menu := application.NewMenu()
	for _, it := range traymenu.BuildItems(s) {
		if it.Separator {
			menu.AddSeparator()
			continue
		}
		item := menu.Add(it.Label)
		item.SetEnabled(it.Enabled)
		id := it.ID
		item.OnClick(func(*application.Context) { c.handleClick(id) })
	}
	c.tray.SetMenu(menu)
}

// handleClick routes a menu item. show/quit are handled here; everything else
// is forwarded to the frontend, which owns the start/stop orchestration.
func (c *trayController) handleClick(id string) {
	switch id {
	case "tray:show":
		showMainWindow(c.app)
	case "tray:quit":
		// Surface the window so the confirm modal is visible even if the app
		// was tray-only, then let the frontend drive the quit flow.
		showMainWindow(c.app)
		c.app.Event.Emit("app:quit-requested")
	default:
		if strings.HasPrefix(id, "tray:") {
			c.app.Event.Emit(id)
		}
	}
}

// showMainWindow brings the main window back from hidden/minimized and
// focuses it.
func showMainWindow(app *application.App) {
	w, ok := app.Window.GetByName(mainWindowName)
	if !ok {
		return
	}
	if ww, ok := w.(*application.WebviewWindow); ok {
		ww.UnMinimise()
		ww.Show()
		ww.Focus()
	}
}
