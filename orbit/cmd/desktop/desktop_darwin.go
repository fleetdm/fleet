package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

//go:embed fleet-logo.svg
var iconDark []byte

func blockWaitForStopEvent(_ string) error {
	log.Debug().Msg("communication channel helpers are not implemented for this platform")
	return nil
}

func trayIconExists() bool {
	log.Debug().Msg("tray icon checker is not implemented for this platform")
	return true
}

// macOSMajorVersion returns the major version of macOS (e.g. 26 for macOS Tahoe).
func macOSMajorVersion() int {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return 0
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), ".", 2)
	v, _ := strconv.Atoi(parts[0])
	return v
}

// resetControlCenter restarts ControlCenter to clear stale NSStatusItem FBSScene state.
//
// On macOS 26 (Tahoe), ControlCenter adopted the FrontBoard/FBSScene architecture for status
// bar items. When fleet-desktop is killed and restarted by orbit, the old FBSScene lingers in a
// "reconnecting" state and the tray icon never reappears. Restarting ControlCenter forces a fresh
// scene to be created on the next NSStatusItem allocation.
//
// This is a no-op on macOS versions before 26.
func resetControlCenter() {
	if macOSMajorVersion() < 26 {
		return
	}
	log.Debug().Msg("macOS 26+: restarting ControlCenter to clear stale status bar scene")
	if err := exec.Command("killall", "ControlCenter").Run(); err != nil {
		log.Debug().Err(err).Msg("killall ControlCenter failed")
		return
	}
	// Wait for ControlCenter to fully restart before systray.Run creates the NSStatusItem.
	time.Sleep(2 * time.Second)
	log.Debug().Msg("ControlCenter restarted")
}

// promptMenuBarAccess opens System Settings to the Control Center / Menu Bar section so the user
// can enable the Fleet Desktop status item.
//
// On macOS 26 (Tahoe), new status bar items are blocked by default and require explicit user
// approval via System Settings. A flag file is written after the first prompt to avoid repeating
// it on every restart.
//
// This is a no-op on macOS versions before 26.
func promptMenuBarAccess() {
	if macOSMajorVersion() < 26 {
		return
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get user home directory for menu bar access prompt")
		return
	}
	appSupportDir := filepath.Join(homeDir, "Library", "Application Support", "com.fleetdm.desktop")
	flagFile := filepath.Join(appSupportDir, ".menubar-prompted")
	if _, err := os.Stat(flagFile); err == nil {
		return // already prompted
	}
	log.Info().Msg("macOS 26+: opening System Settings for menu bar access")
	if err := exec.Command("open", "x-apple.systempreferences:com.apple.ControlCenter-Settings.extension").Run(); err != nil {
		log.Debug().Err(err).Msg("failed to open System Settings for menu bar access")
	}
	if err := os.MkdirAll(appSupportDir, 0o755); err != nil {
		log.Debug().Err(err).Msg("failed to create application support directory")
		return
	}
	f, err := os.Create(flagFile)
	if err != nil {
		log.Debug().Err(err).Msg("failed to write menu bar access prompt flag file")
		return
	}
	f.Close()
}
