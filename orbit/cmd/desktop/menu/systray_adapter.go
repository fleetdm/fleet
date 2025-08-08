package menu

import "fyne.io/systray"

// SystrayFactory implements Factory interface for the actual systray
type SystrayFactory struct{}

// Ensure SystrayFactory implements Factory
var _ Factory = &SystrayFactory{}

// NewSystrayFactory creates a new systray factory
func NewSystrayFactory() *SystrayFactory {
	return &SystrayFactory{}
}

// AddMenuItem creates a new menu item using systray
func (s *SystrayFactory) AddMenuItem(title string, tooltip string) Item {
	return &SystrayMenuItem{
		MenuItem: systray.AddMenuItem(title, tooltip),
	}
}

// AddSeparator adds a separator to the menu
func (s *SystrayFactory) AddSeparator() {
	systray.AddSeparator()
}

// SystrayMenuItem is a thin wrapper around systray.MenuItem that adds the ClickedCh method
// This is needed because systray.MenuItem has ClickedCh as a field, not a method
type SystrayMenuItem struct {
	*systray.MenuItem
}

// ClickedCh returns the channel that's notified when the menu item is clicked
func (s *SystrayMenuItem) ClickedCh() <-chan struct{} {
	return s.MenuItem.ClickedCh
}