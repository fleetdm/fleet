package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/rs/zerolog/log"
)

const (
	appIndicatorExtensionUUID = "appindicatorsupport@rgcjonas.gmail.com"
)

//go:embed icon_dark.png
var iconDark []byte

func blockWaitForStopEvent(channelId string) error {
	return nil
}

func installExtensions() error {
	return installAppIndicatorGNOMEShellExtension()
}

func getOSReleaseID() (string, error) {
	// NOTE: osquery uses `/etc/os-release` to populate the `os_version` table.
	osRelease, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	var id string
	scanner := bufio.NewScanner(bytes.NewReader(osRelease))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.TrimSpace(strings.TrimPrefix(line, "ID="))
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}
	if id == "" {
		return "", osReleaseIDNotFound
	}
	return id, nil
}

var osReleaseIDNotFound = errors.New("/etc/os-release ID not found")

func installAppIndicatorGNOMEShellExtension() error {
	osReleaseID, err := getOSReleaseID()
	if err != nil {
		return fmt.Errorf("get /etc/os-release ID: %w", err)
	}
	if osReleaseID != "fedora" && osReleaseID != "debian" {
		// Nothing to do if this is not debian or fedora. Those are the currently
		// known (supported) distros that need the extension for the tray icon.
		return nil
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect session bus: %w", err)
	}
	defer conn.Close()

	o := conn.Object("org.gnome.Shell.Extensions", "/org/gnome/Shell/Extensions")

	c := o.Call("org.gnome.Shell.Extensions.ListExtensions", 0)
	if c.Err != nil {
		return fmt.Errorf("list remote extensions call: %w", c.Err)
	}

	var extensions map[string]map[string]dbus.Variant
	if err := c.Store(&extensions); err != nil {
		return fmt.Errorf("list remote extensions parse response: %w", err)
	}

	installed := false
	for extension, info := range extensions {
		if extension == appIndicatorExtensionUUID {
			enabled, ok := info["enabled"].Value().(bool)
			if ok && enabled {
				// Nothing to do, extension is installed and enabled.
				log.Debug().Msg("appindicator extension already installed and enabled")
				return nil
			}
			// Extension is installed but not enabled.
			installed = true
			break
		}
	}

	// Extension is not installed, or installed but disabled, so let's install it and enable it.
	if installed {
		log.Info().Msg("appindicator extension installed")
	} else {
		log.Info().Msg("installing appindicator extension...")
		c = o.Call("org.gnome.Shell.Extensions.InstallRemoteExtension", 0, appIndicatorExtensionUUID)
		if c.Err != nil {
			return fmt.Errorf("install remote extension call: %w", c.Err)
		}
		var response string
		if err := c.Store(&response); err != nil {
			return fmt.Errorf("list remote extensions parse response: %w", err)
		}
		switch response {
		case "cancelled":
			// TODO(lucas): Decide what to do if the user cancels. To not ask over and over...
		case "successful":
			// OK
		default:
			log.Info().Msg("unknown response, assuming successful...")
		}
	}
	log.Debug().Msg("enabling appindicator extension...")
	c = o.Call("org.gnome.Shell.Extensions.EnableExtension", 0, appIndicatorExtensionUUID)
	if c.Err != nil {
		return fmt.Errorf("enable extension call: %w", c.Err)
	}

	return nil
}
