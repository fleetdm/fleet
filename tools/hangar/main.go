package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/fleetdm/fleet/tools/hangar/internal/applog"
	"github.com/fleetdm/fleet/tools/hangar/internal/buildinfo"
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
	// `--version` answers "which build is this?" without launching the GUI —
	// the only way to check a bundle someone handed you:
	//   "/Applications/Fleet Hangar.app/Contents/MacOS/fleet-hangar" --version
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("Fleet Hangar", buildinfo.Current().Summary())
		return
	}

	// Bootstrap logging, replaced by the app log as soon as we know where it
	// goes. Only the two failures below can happen before that.
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

	// From here on everything — including whatever the Go runtime prints on
	// its way out if this process dies unexpectedly — lands in hangar.log.
	session := applog.Setup(logDir)
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

	// Fold the managed processes into each heartbeat: when Hangar goes away on
	// its own, this is what says what was running at the time — and whether
	// the fleet server someone was mid-test against went with it.
	session.SetStats(func() []any {
		var running []string
		for _, p := range pm.ListProcesses() {
			if p.State == "running" || p.State == "stopping" {
				running = append(running, p.ID)
			}
		}
		return []any{"running_procs", strings.Join(running, ",")}
	})

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
				// AppKit terminates the process inside this call (app.Quit()
				// is [NSApp terminate:]), so app.Run() never returns and no
				// deferred cleanup in main runs. This is the last Go code on
				// the way out, and so the only place a deliberate exit can be
				// recorded — without it every quit reads as a crash.
				session.Close("user quit")
				return true
			}
			slog.Info("quit requested; asking the frontend to confirm")
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
		// Worth a line: "Hangar disappeared" is also what hiding to the tray
		// looks like to someone who didn't mean to close the window.
		slog.Info("window closed; hiding to tray")
	})

	// macOS dock-icon click while no window is visible → bring it back.
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		showMainWindow(app)
	})

	tray = newTrayController(app, trayIcon)
	logLifecycle(app)

	slog.Info("running")
	if err := app.Run(); err != nil {
		slog.Error("application exited with error", "err", err)
		session.Close("run error: " + err.Error())
		os.Exit(1)
	}
	// Reached only if the event loop stops without AppKit terminating us.
	session.Close("event loop returned")
}
