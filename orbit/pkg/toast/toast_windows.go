//go:build windows

package toast

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const powerShellTimeout = 30 * time.Second

// Show posts the toast for the logged-in user. It must run in the user's session, so Fleet Desktop calls it, not orbit.
func Show(n Notification) error {
	env, err := showEnv(n, appID())
	if err != nil {
		return err
	}
	return runPowerShell(showScript, env)
}

// Remove clears the toast with this tag and group from Notification Center.
func Remove(tag, group string) error {
	return runPowerShell(removeScript, removeEnv(tag, group))
}

// RegisterFleetDesktopAppID registers Fleet Desktop's AppUserModelID so its toasts are labelled "Fleet Desktop" with its
// icon. Windows reads IconUri as an image file path, so the icon is written to iconPath first. Orbit calls this as SYSTEM
// at every start, so the file and values are only written when they differ.
func RegisterFleetDesktopAppID(iconPath string, icon []byte) error {
	if current, err := os.ReadFile(iconPath); err != nil || !bytes.Equal(current, icon) {
		// The shell reads the icon as the logged-in user, so it has to be readable by everyone.
		if err := os.WriteFile(iconPath, icon, 0o644); err != nil { //nolint:gosec
			return fmt.Errorf("writing the Fleet Desktop icon: %w", err)
		}
	}

	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, appIDKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening the AppUserModelID key: %w", err)
	}
	defer k.Close()
	for _, value := range []struct{ name, data string }{
		{name: "DisplayName", data: "Fleet Desktop"},
		{name: "IconUri", data: iconPath},
	} {
		if current, _, err := k.GetStringValue(value.name); err == nil && current == value.data {
			continue
		}
		if err := k.SetExpandStringValue(value.name, value.data); err != nil {
			return fmt.Errorf("setting AppUserModelID value %s: %w", value.name, err)
		}
	}
	return nil
}

const appIDKeyPath = `Software\Classes\AppUserModelId\` + FleetDesktopAppID

// appID picks Fleet Desktop's AppUserModelID when orbit has registered it. A toast posted under an AppUserModelID Windows
// does not know may never appear, so the fallback is one that always exists.
func appID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, appIDKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return powerShellAppID
	}
	k.Close()
	return FleetDesktopAppID
}

func runPowerShell(script string, env []string) error {
	// The absolute path avoids picking up a powershell.exe planted earlier on PATH, and PowerShell 7 cannot load the WinRT types.
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		return fmt.Errorf("locating the system directory: %w", err)
	}
	powerShell := filepath.Join(systemDir, `WindowsPowerShell\v1.0\powershell.exe`)

	ctx, cancel := context.WithTimeout(context.Background(), powerShellTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, powerShell,
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodeCommand(script))
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if out, err := cmd.CombinedOutput(); err != nil {
		const maxOutput = 1000
		output := strings.TrimSpace(string(out))
		if len(output) > maxOutput {
			output = output[:maxOutput]
		}
		return fmt.Errorf("running PowerShell: %w: %s", err, output)
	}
	return nil
}
