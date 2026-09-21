//go:build windows

package mdmsecret

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"golang.org/x/sys/windows/registry"
)

// Fleet already owns SOFTWARE\FleetDM\Orbit (the installer records the install path there), so the
// delivered secret lives alongside it rather than in a new hive. The MSI creates the key with a DACL
// restricted to SYSTEM and Administrators, matching the ACL it puts on secret.txt, so a non-admin
// local user cannot read a secret that has not been adopted yet.
const (
	keyPath   = `SOFTWARE\FleetDM\Orbit`
	valueName = "EnrollSecret"
)

// Read returns the enroll secret Fleet MDM delivered to this device, or ErrNotFound when none is
// waiting. It never puts the value in an error, so a returned error is safe to log.
func Read() (string, error) {
	return read(registry.LOCAL_MACHINE, keyPath)
}

// Clear removes the delivered secret. It is called once orbit has adopted the value, so the window
// in which an unconsumed secret sits readable on disk is as short as orbit can make it. Clearing an
// already-absent value is not an error.
func Clear() error {
	return clearSecret(registry.LOCAL_MACHINE, keyPath)
}

// read and clearSecret take the root and path explicitly so tests can exercise them against a key
// they can actually create; writing under HKLM needs elevation that a test process may not have.
func read(root registry.Key, path string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer key.Close()

	secret, _, err := key.GetStringValue(valueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read %s from %s: %w", valueName, path, err)
	}

	secret = strings.TrimSpace(secret)
	// The installer seeds unset properties with "dummy"; treat that, and an empty value, as nothing
	// waiting rather than handing an unusable secret to the enroll path.
	if secret == "" || secret == constant.UnusedFlagKeyword {
		return "", ErrNotFound
	}
	return secret, nil
}

func clearSecret(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s for write: %w", path, err)
	}
	defer key.Close()

	if err := key.DeleteValue(valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete %s from %s: %w", valueName, path, err)
	}
	return nil
}
