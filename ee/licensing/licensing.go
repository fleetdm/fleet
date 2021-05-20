package licensing

import (
	"github.com/fleetdm/fleet/server/kolide"
)

func LoadLicense(licenseKey string) (*kolide.LicenseInfo, error) {
	// TODO actual logic here

	if licenseKey == "" {
		return &kolide.LicenseInfo{Tier: "core"}, nil
	}

	return &kolide.LicenseInfo{Tier: "basic"}, nil
}
