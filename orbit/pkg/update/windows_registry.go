//go:build windows

package update

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	REG_FLEETD_DISPLAY_NAME = "Fleet osquery"
	// registry paths, absolute and relative to the HKEY_LOCAL_MACHINE root key - see
	// https://pkg.go.dev/golang.org/x/sys/windows/registry#LOCAL_MACHINE and
	// https://learn.microsoft.com/en-us/troubleshoot/windows-server/performance/windows-registry-advanced-users
	HKEY_LOCAL_MACHINE_PATH = `Computer\HKEY_LOCAL_MACHINE`
	REG_UNINSTALL_REL_PATH  = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
	REG_UNINSTALL_ABS_PATH  = HKEY_LOCAL_MACHINE_PATH + `\` + REG_UNINSTALL_REL_PATH
)

func updateUninstallFleetdRegistryVersion(newVersion string) error {
	// Since fleetd doesn't know its GUID key in the registry, iterate through all of them until we find
	// the appropriate key

	// path from the HKEY_LOCAL_MACHINE registry root key ("Computer\HKEY_LOCAL_MACHINE") to the key
	// for the uninstall Fleetd registry entry. That is, REG_UNINSTALL_REL_PATH + `\` + (GUID of
	// Fleetd entry ). This format is for compatibility with the registry.OpenKey function signature
	uninstallFleetdRegRelPath, err := findUninstallFleetdRegKeyRelPath()
	if err != nil {
		return fmt.Errorf(`couldn't find the uninstall fleetd registry key in '%v': %w`, REG_UNINSTALL_ABS_PATH, err)
	}

	setKey, err := registry.OpenKey(registry.LOCAL_MACHINE, uninstallFleetdRegRelPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf(`couldn't open 'SET_VALUE' key handle for '%v\%v": %w`, HKEY_LOCAL_MACHINE_PATH, uninstallFleetdRegRelPath, err)
	}
	defer setKey.Close()

	if err := setKey.SetStringValue("DisplayVersion", newVersion); err != nil {
		return fmt.Errorf(`couldn't set value 'DisplayVersion' for '%v\%v: %w`, HKEY_LOCAL_MACHINE_PATH, uninstallFleetdRegRelPath, err)
	}
	return nil
}

func findUninstallFleetdRegKeyRelPath() (string, error) {
	// get the existing keys in the Uninstall registry directory
	enumerateKeyHandle, err := registry.OpenKey(registry.LOCAL_MACHINE, REG_UNINSTALL_REL_PATH, registry.READ)
	if err != nil {
		return "", fmt.Errorf(`couldn't open registry key '%v': %w`, REG_UNINSTALL_ABS_PATH, err)
	}
	defer enumerateKeyHandle.Close()

	stat, err := enumerateKeyHandle.Stat()
	if err != nil {
		return "", fmt.Errorf(`couldn't get stat from registry key handle for '%v': %w`, REG_UNINSTALL_ABS_PATH, err)
	}
	subKeyCount := stat.SubKeyCount

	keys, err := enumerateKeyHandle.ReadSubKeyNames(int(subKeyCount))
	if err != nil {
		return "", fmt.Errorf(`couldn't read subkeys of registry key handle for '%v': %w`, REG_UNINSTALL_ABS_PATH, err)
	}

	// find the Fleetd entry in the existing keys
	var fleetdKey string
	for _, key := range keys {
		keyHandle, err := registry.OpenKey(registry.LOCAL_MACHINE, REG_UNINSTALL_REL_PATH+`\`+key, registry.READ)
		if err != nil {
			return "", fmt.Errorf(`couldn't open registry subkey handle for '%v\%v': %w`, REG_UNINSTALL_ABS_PATH, key, err)
		}
		defer keyHandle.Close()
		displayName, _, err := keyHandle.GetStringValue("DisplayName")
		if err != nil {
			if errors.Is(err, registry.ErrNotExist) {
				// this key doesn't have a `DisplayName`, so it's not the entry for Fleetd - keep looking
				continue
			}
			return "", fmt.Errorf(`couldn't get registry string value 'DisplayName' for '%v\%v': %w`, REG_UNINSTALL_ABS_PATH, key, err)
		}
		if displayName == REG_FLEETD_DISPLAY_NAME {
			fleetdKey = key
			break
		}
	}

	if fleetdKey == "" {
		return "", fmt.Errorf(`couldn't find a corresponding registry value for fleetd in 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`)
	}

	return REG_UNINSTALL_REL_PATH + `\` + fleetdKey, nil
}
