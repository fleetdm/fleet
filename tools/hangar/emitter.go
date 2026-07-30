package main

import "github.com/wailsapp/wails/v3/pkg/application"

// wailsEmitter adapts the process engine's Emitter interface to Wails'
// event bus. The app pointer is set right after application.New (events only
// fire at runtime), which sidesteps the app↔services construction cycle.
type wailsEmitter struct {
	app *application.App
}

func (e *wailsEmitter) Emit(name string, data any) {
	if e.app != nil {
		e.app.Event.Emit(name, data)
	}
}
