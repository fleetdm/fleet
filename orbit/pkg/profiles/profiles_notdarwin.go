//go:build !darwin
// +build !darwin

package profiles

import "github.com/fleetdm/fleet/v4/server/fleet"

func GetFleetdConfig() (*fleet.MDMAppleFleetdConfiguration, error) {
	return nil, ErrNotImplemented
}
