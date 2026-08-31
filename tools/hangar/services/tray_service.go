package services

import "github.com/fleetdm/fleet/tools/hangar/internal/traymenu"

// TrayService lets the frontend push tray state (branch + service health),
// which rebuilds the native tray menu. Mirrors update_tray from tray.rs.
type TrayService struct {
	update func(traymenu.State)
}

// NewTrayService wires the service to the tray-rebuild callback.
func NewTrayService(update func(traymenu.State)) *TrayService {
	return &TrayService{update: update}
}

// UpdateTray rebuilds the tray menu from the given state.
func (s *TrayService) UpdateTray(state traymenu.State) {
	if s.update != nil {
		s.update(state)
	}
}
