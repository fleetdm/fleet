package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"

	"github.com/fleetdm/fleet/v4/orbit/pkg/user"
	"github.com/godbus/dbus/v5"
	"github.com/rs/zerolog/log"
)

//go:embed icon_dark.png
var iconDarkDefault []byte

//go:embed icon_kde.png
var iconKDE []byte

var iconDark = getIcon()

func getIcon() []byte {
	if isKDE() {
		return iconKDE
	}
	return iconDarkDefault
}

// isKDE reports whether the user with the active GUI session is running KDE
// Plasma process.
func isKDE() bool {
	guiUser, err := user.LoggedInGuiUser()
	if err != nil {
		log.Debug().Err(err).Msg("isKDE: look up logged-in GUI user")
		return false
	}
	if guiUser == nil {
		return false
	}
	uid := strconv.FormatInt(guiUser.ID, 10)
	if err := exec.Command("pgrep", "-u", uid, "-x", "plasmashell").Run(); err != nil {
		return false
	}
	return true
}

func blockWaitForStopEvent(_ string) error {
	log.Debug().Msg("communication channel helpers are not implemented for this platform")
	return nil
}

func trayIconExists() bool {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Error().Err(err)
	}

	// Get the name we would expect systray to reserve for our tray icon.
	trayIconDbusName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	// Get the names this session currently owns.
	ownedDbusNames := conn.Names()
	// If the tray icon name isn't in the list, it likely means the tray icon is
	// no longer visible.
	return slices.Contains(ownedDbusNames, trayIconDbusName)
}
