//go:build windows

package profiles

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"golang.org/x/sys/windows/registry"
)

// On Windows MDM the enroll secret is delivered by a Fleet-managed configuration profile that writes it to the registry. orbit
// loads the value and then clears it, so the presence of the value means "a secret is waiting" and its absence means orbit
// already took it. That makes one channel serve both first enrollment and recovery: an administrator resends the profile (with a
// new secret) and the value reappears.
//
// Fleet already owns SOFTWARE\FleetDM\Orbit (the installer records the install path there), so the delivered secret lives
// alongside it rather than in a new hive.
const (
	enrollSecretKeyPath   = `SOFTWARE\FleetDM\Orbit`
	enrollSecretValueName = "EnrollSecret"
)

// GetEnrollSecret returns the enroll secret Fleet MDM delivered to this device, or ErrEnrollSecretNotFound when none is waiting.
func GetEnrollSecret() (string, error) {
	return getEnrollSecret(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// ClearEnrollSecret removes the delivered secret. It is called once orbit has loaded the value.
func ClearEnrollSecret() error {
	return clearEnrollSecret(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// getEnrollSecret and clearEnrollSecret take the root and path explicitly so tests can exercise them.
func getEnrollSecret(root registry.Key, path string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrEnrollSecretNotFound
		}
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer key.Close()

	secret, _, err := key.GetStringValue(enrollSecretValueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrEnrollSecretNotFound
		}
		return "", fmt.Errorf("read %s from %s: %w", enrollSecretValueName, path, err)
	}

	secret = strings.TrimSpace(secret)
	// The installer seeds unset properties with "dummy"; treat that, and an empty value, as nothing waiting.
	if secret == "" || secret == constant.UnusedFlagKeyword {
		return "", ErrEnrollSecretNotFound
	}
	return secret, nil
}

func clearEnrollSecret(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s for write: %w", path, err)
	}
	defer key.Close()

	if err := key.DeleteValue(enrollSecretValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete %s from %s: %w", enrollSecretValueName, path, err)
	}
	return nil
}
