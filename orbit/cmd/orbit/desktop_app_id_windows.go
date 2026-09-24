//go:build windows

package main

import (
	_ "embed"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/toast"
	"github.com/rs/zerolog/log"
)

// fleetDesktopIcon is the same icon fleet-desktop.exe embeds.
//
//go:embed windows_app.ico
var fleetDesktopIcon []byte

// registerFleetDesktopAppID gives Fleet Desktop's toasts their name and icon.
func registerFleetDesktopAppID(rootDir string) {
	if err := toast.RegisterFleetDesktopAppID(filepath.Join(rootDir, "fleet-desktop.ico"), fleetDesktopIcon); err != nil {
		log.Error().Err(err).Msg("could not register the Fleet Desktop notification identity")
	}
}
