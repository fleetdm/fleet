//go:build windows

package update

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
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

// renameUninstallFleetdRegistryEntry sets the name Installed apps shows for fleetd. It leaves an entry with another
// publisher alone: renaming it would hide it from findUninstallFleetdRegKeyRelPath, which matches the new name only with
// Fleet's publisher.
func renameUninstallFleetdRegistryEntry(name string) error {
	uninstallFleetdRegRelPath, err := findUninstallFleetdRegKeyRelPath()
	if err != nil {
		return fmt.Errorf(`couldn't find the uninstall fleetd registry key in '%v': %w`, REG_UNINSTALL_ABS_PATH, err)
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, uninstallFleetdRegRelPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf(`couldn't open key handle for '%v\%v': %w`, HKEY_LOCAL_MACHINE_PATH, uninstallFleetdRegRelPath, err)
	}
	defer key.Close()
	if current, _, err := key.GetStringValue("DisplayName"); err == nil && current == name {
		return nil
	}
	if publisher, _, _ := key.GetStringValue("Publisher"); publisher != fleetdPublisher {
		return fmt.Errorf(`not renaming '%v\%v': publisher is %q`, HKEY_LOCAL_MACHINE_PATH, uninstallFleetdRegRelPath, publisher)
	}
	if err := key.SetStringValue("DisplayName", name); err != nil {
		return fmt.Errorf(`couldn't set value 'DisplayName' for '%v\%v': %w`, HKEY_LOCAL_MACHINE_PATH, uninstallFleetdRegRelPath, err)
	}
	return nil
}

// restoreUninstallFleetdRegistryName gives fleetd's Installed apps entry back the installer's name.
func restoreUninstallFleetdRegistryName() error {
	return renameUninstallFleetdRegistryEntry(fleetdInstallerName)
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
		displayName, publisher, err := readUninstallEntryName(REG_UNINSTALL_REL_PATH + `\` + key)
		if err != nil {
			// Another app's unreadable or half-removed entry mustn't hide fleetd's.
			continue
		}
		if !isFleetdUninstallEntry(displayName, publisher) {
			continue
		}
		// An old uninstall script can leave a renamed entry behind without the product's Windows Installer
		// registration, so the installed product's entry wins over the first match.
		if isInstalledProduct(key) {
			fleetdKey = key
			break
		}
		if fleetdKey == "" {
			fleetdKey = key
		}
	}

	if fleetdKey == "" {
		return "", errors.New(`couldn't find a corresponding registry value for fleetd in 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`)
	}

	return REG_UNINSTALL_REL_PATH + `\` + fleetdKey, nil
}

// isInstalledProduct returns whether an Installed apps entry's key is the product code of a product that Windows Installer
// has registered per machine, as fleetd's MSI installs.
func isInstalledProduct(key string) bool {
	packed, ok := packedProductCode(key)
	if !ok {
		return false
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Classes\Installer\Products\`+packed, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// readUninstallEntryName returns an Installed apps entry's DisplayName and Publisher.
func readUninstallEntryName(relPath string) (displayName, publisher string, err error) {
	keyHandle, err := registry.OpenKey(registry.LOCAL_MACHINE, relPath, registry.READ)
	if err != nil {
		return "", "", err
	}
	defer keyHandle.Close()
	displayName, _, err = keyHandle.GetStringValue("DisplayName")
	if err != nil {
		return "", "", err
	}
	// A missing or unreadable Publisher only rules out the renamed entry, so it isn't an error.
	publisher, _, _ = keyHandle.GetStringValue("Publisher")
	return displayName, publisher, nil
}
