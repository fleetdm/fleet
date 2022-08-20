//go:build windows
// +build windows

package main

import (
	_ "embed"
	"fmt"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// For Windows we must use ico format for the icon,
// see https://github.com/getlantern/systray/blob/6065fda28be8c8d91aeb5e20de25e1600b8664a3/systray_windows.go#L850-L856.

//go:embed icon_light.ico
var iconLight []byte

//go:embed icon_dark.ico
var iconDark []byte

// Adapted from MIT licensed code in
// https://github.com/WireGuard/wireguard-go/commit/7e962a9932667f4a161b20aba5ff1c75ab8e578a
// and https://gist.github.com/jerblack/1d05bbcebb50ad55c312e4d7cf1bc909

// mkwinsyscall generates the Window syscall from the //sys comment below.
//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output generated_syscall_windows.go desktop_windows.go
//sys regNotifyChangeKeyValue(key windows.Handle, watchSubtree bool, notifyFilter uint32, event windows.Handle, asynchronous bool) (regerrno error) = advapi32.RegNotifyChangeKeyValue

const (
	// Registry path where theme value is stored.
	registryPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	registryKey  = "AppsUseLightTheme"
	// REG_NOTIFY_CHANGE_LAST_SET notifies the caller of changes to a value of the key. This can include adding or deleting a value, or changing an existing value.
	REG_NOTIFY_CHANGE_LAST_SET uint32 = 0x00000004
)

func getSystemTheme() (theme, error) {
	// Adapted from https://stackoverflow.com/a/58494769/491710
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return themeDark, err
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue(registryKey)
	if err != nil {
		return themeDark, err
	}

	switch val {
	case 0:
		return themeDark, nil
	case 1:
		return themeLight, nil
	default:
		return themeUnknown, fmt.Errorf("unknown theme value %d", val)
	}
}

func watchSystemTheme(iconManager *iconManager) {
	for {
		// Function call for proper defer semantics.
		func() {
			// Open the key within the loop, because "If the specified key is closed, the event is
			// signaled. This means that an application should not depend on the key being open after
			// returning from a wait operation on the event." -
			// https://docs.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regnotifychangekeyvalue
			key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, syscall.KEY_NOTIFY)
			if err != nil {
				fmt.Println("open key: " + err.Error())
				return
			}
			defer key.Close()

			err = regNotifyChangeKeyValue(windows.Handle(key), false, REG_NOTIFY_CHANGE_LAST_SET, windows.Handle(0), false)
			if err != nil {
				fmt.Println("Setting up change notification on registry value failed: %v", err)
			}

			theme, err := getSystemTheme()
			if err != nil {
				log.Error().Err(err).Msg("get system theme")
				return
			}
			log.Debug().Str("theme", string(theme)).Msg("got theme update")

			// The systray library has a timeout issue trying to change the icon sometimes. As a
			// cheap workaround, do this update a handful of times hoping one will work. Sadly the
			// API doesn't return an error in these cases.
			for i := 0; i < 10; i++ {
				iconManager.UpdateTheme(theme)
				time.Sleep(1 * time.Second)
			}
		}()
	}
}
