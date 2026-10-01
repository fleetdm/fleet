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
	if !appIDRegistered() {
		return ErrAppIDNotRegistered
	}
	env, err := showScriptEnv(n)
	if err != nil {
		return err
	}
	return runPowerShell(showScript, env)
}

// Remove clears the toast with this tag and group from Notification Center.
func Remove(tag, group string) error {
	if !appIDRegistered() {
		// Nothing was ever posted, because Show refuses without the identity.
		return nil
	}
	if err := validateTagAndGroup(tag, group); err != nil {
		return err
	}
	return runPowerShell(removeScript, removeScriptEnv(tag, group))
}

// RegisterFleetDesktopAppID registers Fleet Desktop's AppUserModelID so its toasts are labeled "Fleet Desktop" with its
// icon. Windows reads IconUri as an image file path, so the icon is written to iconPath first. Orbit calls this as SYSTEM
// at every start, so the file and values are only written when they differ.
func RegisterFleetDesktopAppID(iconPath string, icon []byte) error {
	if current, err := os.ReadFile(iconPath); err != nil || !bytes.Equal(current, icon) {
		if err := writeIcon(iconPath, icon); err != nil {
			return err
		}
	}

	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, appIDKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening the AppUserModelID key: %w", err)
	}
	defer k.Close()
	for _, value := range []struct{ name, data string }{
		{name: "DisplayName", data: fleetDesktopDisplayName},
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

const (
	appIDKeyPath            = `Software\Classes\AppUserModelId\` + FleetDesktopAppID
	fleetDesktopDisplayName = "Fleet Desktop"
)

// writeIcon replaces the icon in one step, so the shell never reads a half-written file.
func writeIcon(iconPath string, icon []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(iconPath), filepath.Base(iconPath)+".*")
	if err != nil {
		return fmt.Errorf("creating the Fleet Desktop icon: %w", err)
	}
	defer os.Remove(temp.Name())
	// The shell reads the icon as the logged-in user, so it has to be readable by everyone.
	if err := temp.Chmod(0o644); err != nil { //nolint:gosec
		return fmt.Errorf("setting permissions on the Fleet Desktop icon: %w", err)
	}
	if _, err := temp.Write(icon); err != nil {
		return fmt.Errorf("writing the Fleet Desktop icon: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("closing the Fleet Desktop icon: %w", err)
	}
	return os.Rename(temp.Name(), iconPath)
}

// appIDRegistered reports whether orbit has registered Fleet Desktop's AppUserModelID.
func appIDRegistered() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, appIDKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

func runPowerShell(script string, env []string) error {
	// The absolute path avoids picking up a powershell.exe planted earlier on PATH, and PowerShell 7 cannot load the WinRT types.
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		return fmt.Errorf("locating the system directory: %w", err)
	}
	// Canonical path for PowerShell
	powerShell := filepath.Join(systemDir, `WindowsPowerShell\v1.0\powershell.exe`)

	ctx, cancel := context.WithTimeout(context.Background(), powerShellTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, powerShell,
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodeCommand(script))
	cmd.Env = childEnv(os.Environ(), env)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	// Without this, PowerShell exiting without closing its pipes would block CombinedOutput past the context timeout, and
	// the caller holds a lock while it waits.
	cmd.WaitDelay = time.Second
	if out, err := cmd.CombinedOutput(); err != nil {
		const maxOutput = 1000
		output := strings.TrimSpace(string(out))
		if len(output) > maxOutput {
			output = strings.ToValidUTF8(output[:maxOutput], "")
		}
		return fmt.Errorf("running PowerShell: %w: %s", err, output)
	}
	return nil
}
