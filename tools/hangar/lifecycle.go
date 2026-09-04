package main

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// logLifecycle records the macOS application events that bracket the stretches
// of time Hangar has been reported to disappear in: system and display
// sleep/wake, and the screen-parameter change that Wails' own screen-cache
// refresh has been observed dying on (wailsapp/wails#5556 — it reads
// autoreleased NSString buffers after the pool has drained, which surfaces as
// an unrecoverable "fatal error: invalid pointer found on stack").
//
// If a session's log ends immediately after one of these lines, that's the
// answer; internal/applog captures the runtime's dump right below it.
func logLifecycle(app *application.App) {
	// A hook runs synchronously, before the event's listeners — one of which
	// is the suspect refresh. A plain listener would be dispatched in its own
	// goroutine, racing the crash, and might never be scheduled to write this
	// line. Note this only observes: cancelling the event here would stop
	// Wails from refreshing its screen cache at all.
	app.Event.RegisterApplicationEventHook(
		events.Mac.ApplicationDidChangeScreenParameters,
		func(*application.ApplicationEvent) {
			slog.Info("screen parameters changed; wails refreshes its screen cache next")
		},
	)

	for _, e := range []struct {
		id  events.ApplicationEventType
		msg string
	}{
		{events.Mac.ApplicationWillSleep, "system going to sleep"},
		{events.Mac.ApplicationDidWake, "system woke"},
		{events.Mac.ApplicationScreensDidSleep, "displays slept"},
		{events.Mac.ApplicationScreensDidWake, "displays woke"},
		{events.Mac.ApplicationWillTerminate, "application will terminate"},
	} {
		app.Event.OnApplicationEvent(e.id, func(*application.ApplicationEvent) {
			slog.Info(e.msg)
		})
	}
}
