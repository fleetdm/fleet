//go:build !darwin

package profiles

import "github.com/fleetdm/fleet/v4/server/fleet"

func GetFleetdConfig() (*fleet.MDMAppleFleetdConfig, error) {
	return nil, ErrNotImplemented
}

func IsEnrolledIntoMatchingURL(u string) (bool, error) {
	return false, ErrNotImplemented
}
