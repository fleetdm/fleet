//go:build windows
// +build windows

package main

import (
	_ "embed"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// For Windows we must use ico format for the icon,
// see https://github.com/getlantern/systray/blob/6065fda28be8c8d91aeb5e20de25e1600b8664a3/systray_windows.go#L850-L856.

//go:embed icon_white.ico
var icoBytes []byte

// Adapted from MIT licensed code in
// https://github.com/WireGuard/wireguard-go/commit/7e962a9932667f4a161b20aba5ff1c75ab8e578a
// and https://gist.github.com/jerblack/1d05bbcebb50ad55c312e4d7cf1bc909

// mkwinsyscall generates the Window syscall from the //sys comment below.
//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output generated_syscall_windows.go desktop_windows.go
//sys regNotifyChangeKeyValue(key windows.Handle, watchSubtree bool, notifyFilter uint32, event windows.Handle, asynchronous bool) (regerrno error) = advapi32.RegNotifyChangeKeyValue

const regPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

func getSystemTheme() string {
	// From https://stackoverflow.com/a/58494769/491710

	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return "open key: " + err.Error()
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return "getStringValue: " + err.Error()
	}

	switch val {
	case 0:
		return "dark"
	case 1:
		return "light"
	default:
		return "unknown"
	}
}

const (
	KEY_NOTIFY uint32 = 0x0010 // should be defined upstream as registry.KEY_NOTIFY
)

const (
	// REG_NOTIFY_CHANGE_NAME notifies the caller if a subkey is added or deleted.
	REG_NOTIFY_CHANGE_NAME uint32 = 0x00000001

	// REG_NOTIFY_CHANGE_ATTRIBUTES notifies the caller of changes to the attributes of the key, such as the security descriptor information.
	REG_NOTIFY_CHANGE_ATTRIBUTES uint32 = 0x00000002

	// REG_NOTIFY_CHANGE_LAST_SET notifies the caller of changes to a value of the key. This can include adding or deleting a value, or changing an existing value.
	REG_NOTIFY_CHANGE_LAST_SET uint32 = 0x00000004

	// REG_NOTIFY_CHANGE_SECURITY notifies the caller of changes to the security descriptor of the key.
	REG_NOTIFY_CHANGE_SECURITY uint32 = 0x00000008

	// REG_NOTIFY_THREAD_AGNOSTIC indicates that the lifetime of the registration must not be tied to the lifetime of the thread issuing the RegNotifyChangeKeyValue call. Note: This flag value is only supported in Windows 8 and later.
	REG_NOTIFY_THREAD_AGNOSTIC uint32 = 0x10000000
)

func watchSystemTheme() {
	// theme := getSystemTheme()

	key, err := registry.OpenKey(registry.CURRENT_USER, regPath, syscall.KEY_NOTIFY|registry.QUERY_VALUE)
	if err != nil {
		fmt.Println("open key: " + err.Error())
	}
	defer key.Close()

	for {
		err := regNotifyChangeKeyValue(windows.Handle(key), false, REG_NOTIFY_CHANGE_LAST_SET, windows.Handle(0), false)
		if err != nil {
			fmt.Println("Setting up change notification on registry value failed: %v", err)
		}

		fmt.Println(getSystemTheme())
	}
}
