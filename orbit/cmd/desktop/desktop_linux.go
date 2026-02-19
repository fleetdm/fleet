package main

import (
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/user"
	"github.com/godbus/dbus/v5"
	"github.com/rs/zerolog/log"
)

//go:embed icon_dark.png
var iconDarkDefault []byte

//go:embed icon_dark_kde.png
var iconDarkKDE []byte

var iconDark = getIcon()

func getIcon() []byte {
	if isKDE() {
		return iconDarkKDE
	}
	return iconDarkDefault
}

func isKDE() bool {
	session, err := user.GetLoggedInUserDisplaySession()
	if err != nil {
		return false
	}
	return session != nil && strings.ToLower(session.Desktop) == "kde"
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
