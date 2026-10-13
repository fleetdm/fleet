//go:build windows

package autopilot

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// Windows writes the profile it downloaded from the Autopilot service here, as one JSON string. The ZTDID is not stored
// as a value of its own anywhere else.
const (
	policyCacheKeyPath   = `SOFTWARE\Microsoft\Provisioning\AutopilotPolicyCache`
	policyCacheValueName = "PolicyJsonCache"
)

// RegistrationID returns the device's Autopilot device ID (ZTDID), the GUID Microsoft Graph returns as
// windowsAutopilotDeviceIdentity.id. It returns an empty string when the device has no Autopilot profile.
func RegistrationID() (string, error) {
	return registrationID(registry.LOCAL_MACHINE, policyCacheKeyPath)
}

// registrationID takes the root and path explicitly so tests can exercise it.
func registrationID(root registry.Key, path string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer key.Close()

	policyJSON, _, err := key.GetStringValue(policyCacheValueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read %s from %s: %w", policyCacheValueName, path, err)
	}
	return parseRegistrationID(policyJSON)
}
