package services

import "github.com/wailsapp/wails/v3/pkg/application"

// DialogService exposes native file/folder pickers, replacing the
// @tauri-apps/plugin-dialog `open()` the frontend used. PromptForSingleSelection
// returns "" when the user cancels. Uses application.Get() so it needs no
// app field (which would otherwise require an unbindable setter method).
type DialogService struct{}

// PickFolder opens a directory chooser.
func (s *DialogService) PickFolder() (string, error) {
	d := application.Get().Dialog.OpenFile()
	d.CanChooseDirectories(true)
	d.CanChooseFiles(false)
	return d.PromptForSingleSelection()
}

// PickFile opens a file chooser.
func (s *DialogService) PickFile() (string, error) {
	d := application.Get().Dialog.OpenFile()
	d.CanChooseDirectories(false)
	d.CanChooseFiles(true)
	return d.PromptForSingleSelection()
}

// PickFileWithFilter opens a file chooser limited to one display/pattern
// filter (e.g. "YAML", "*.yml;*.yaml").
func (s *DialogService) PickFileWithFilter(displayName, pattern string) (string, error) {
	d := application.Get().Dialog.OpenFile()
	d.CanChooseDirectories(false)
	d.CanChooseFiles(true)
	d.AddFilter(displayName, pattern)
	return d.PromptForSingleSelection()
}
