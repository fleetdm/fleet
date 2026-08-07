package main

import (
	"embed"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
	"github.com/fleetdm/fleet/tools/hangar/internal/traymenu"
	"github.com/fleetdm/fleet/tools/hangar/services"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/tray-icon.png
var trayIcon []byte

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	logDir, err := paths.LogDir()
	if err != nil {
		slog.Error("resolve log dir", "err", err)
		os.Exit(1)
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		slog.Error("resolve data dir", "err", err)
		os.Exit(1)
	}
	slog.Info("starting Fleet Hangar", "bundleID", paths.BundleID, "logDir", logDir, "dataDir", dataDir)

	// Warm the login-shell PATH (so the first spawn doesn't pay the probe
	// latency; shellpath.Command then resolves tools against it) and reap any
	// orphans a prior crashed session left behind, before the tray/commands
	// come up.
	shellpath.Warm()
	processes.CleanOrphansFromPriorRun(dataDir)

	// intentionalQuit gates hide-to-tray (window close) vs a real quit.
	var intentionalQuit atomic.Bool

	emitter := &wailsEmitter{}
	pm := processes.New(logDir, dataDir, emitter)

	var app *application.App
	var tray *trayController

	app = application.New(application.Options{
		Name:        "Fleet Hangar",
		Description: "Desktop control panel for Fleet contributors",
		Services: []application.Service{
			application.NewService(&services.SettingsService{}),
			application.NewService(services.NewProcessService(pm, func() {
				// Called after ShutdownNow's teardown: flag the exit as
				// intentional so the close hook lets the window go, then quit.
				intentionalQuit.Store(true)
				app.Quit()
			})),
			application.NewService(services.NewScepService(pm)),
			application.NewService(&services.MdmAssetsService{}),
			application.NewService(services.NewTufService(pm)),
			application.NewService(services.NewTrayService(func(s traymenu.State) {
				if tray != nil {
					tray.update(s)
				}
			})),
			application.NewService(&services.GitService{}),
			application.NewService(&services.DBService{}),
			application.NewService(&services.GitopsService{}),
			application.NewService(&services.FleetctlService{}),
			application.NewService(&services.TroubleshootService{}),
			application.NewService(&services.PerfService{}),
			application.NewService(&services.PerfConfigService{}),
			application.NewService(&services.DepsService{}),
			application.NewService(&services.DialogService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Closing the window hides it (we keep running in the tray/dock),
			// so don't terminate when the last window closes.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
			// Regular: keep the dock icon + Cmd+Tab entry. The tray is a
			// secondary status entry. (See tray.rs notes on why we don't flip
			// to Accessory.)
			ActivationPolicy: application.ActivationPolicyRegular,
		},
		// Cmd+Q / app menu Quit / dock-right-click Quit route to the frontend
		// confirm flow, which calls ShutdownNow when the user confirms.
		ShouldQuit: func() bool {
			if intentionalQuit.Load() {
				return true
			}
			app.Event.Emit("app:quit-requested")
			return false
		},
	})
	emitter.app = app

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   mainWindowName,
		Title:  "Fleet Hangar",
		Width:  1280,
		Height: 860,
		URL:    "/",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarDefault,
		},
	})

	// Hide-to-tray: the X / Cmd+W close hides the window but keeps the app
	// alive. A WindowClosing hook that cancels the event skips the default
	// destroy handler. During an intentional quit we let it close.
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if intentionalQuit.Load() {
			return
		}
		e.Cancel()
		win.Hide()
	})

	// macOS dock-icon click while no window is visible → bring it back.
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		showMainWindow(app)
	})

	tray = newTrayController(app, trayIcon)

	slog.Info("running")
	if err := app.Run(); err != nil {
		slog.Error("application exited with error", "err", err)
		os.Exit(1)
	}
}
