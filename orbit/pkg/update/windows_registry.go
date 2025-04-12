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
	// https://pkg.go.dev/golang.org/x/sys/windows/registry#LOCAL_MACHINE and https://learn.microsoft.com/en-us/troubleshoot/windows-server/performance/windows-registry-advanced-users
	REG_REL_PATH = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
	REG_ABS_PATH = `Computer\HKEY_LOCAL_MACHINE` + `\` + REG_REL_PATH
)

func updateRegistryVersion(newVersion string) error {
	// Since fleetd doesn't know its GUID key in the registry, iterate through all of them until we find
	// the appropriate key

	enumerateKey, err := registry.OpenKey(registry.LOCAL_MACHINE, REG_REL_PATH, registry.READ)
	if err != nil {
		return fmt.Errorf(`couldn't open registry key '%v': %w`, REG_ABS_PATH, err)
	}

	stat, err := enumerateKey.Stat()
	if err != nil {
		return fmt.Errorf(`couldn't get stat from registry key handle for '%v': %w`, REG_ABS_PATH, err)
	}
	subKeyCount := stat.SubKeyCount

	keys, err := enumerateKey.ReadSubKeyNames(int(subKeyCount))
	if err != nil {
		return fmt.Errorf(`couldn't read subkeys of registry key handle for '%v': %w`, REG_ABS_PATH, err)
	}
	enumerateKey.Close()

	fleetdRegKey, err := findFleetdRegKey(keys)

	setKey, err := registry.OpenKey(registry.LOCAL_MACHINE, REG_REL_PATH+`\`+fleetdRegKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf(`couldn't open 'SET_VALUE' key handle for '%v\%v": %w`, REG_ABS_PATH, fleetdRegKey, err)
	}
	defer setKey.Close()

	if err := setKey.SetStringValue("DisplayVersion", newVersion); err != nil {
		return fmt.Errorf(`couldn't set value 'DisplayVersion' for '%v\%v: %w`, REG_ABS_PATH, fleetdRegKey, err)
	}
	return nil
}

func findFleetdRegKey(keys []string) (string, error) {
	var fleetdRegKey string
	for _, key := range keys {
		keyHandle, err := registry.OpenKey(registry.LOCAL_MACHINE, REG_REL_PATH+`\`+key, registry.READ)
		if err != nil {
			return "", fmt.Errorf(`couldn't open registry subkey handle for '%v\%v': %w`, REG_ABS_PATH, key, err)
		}
		displayName, _, err := keyHandle.GetStringValue("DisplayName")
		if err != nil {
			if errors.Is(err, registry.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf(`couldn't get registry string value 'DisplayName' for '%v\%v': %w`, REG_ABS_PATH, key, err)
		}
		if displayName == REG_FLEETD_DISPLAY_NAME {
			fleetdRegKey = key
			break
		}
	}

	if fleetdRegKey == "" {
		return "", fmt.Errorf(`couldn't find a corresponding registry value for fleetd in 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`)
	}

	return fleetdRegKey, nil
}
