//go:build !darwin

package profiles

import "github.com/fleetdm/fleet/v4/server/fleet"

func GetFleetdConfig() (*fleet.MDMAppleFleetdConfig, error) {
	return nil, ErrNotImplemented
}

func IsEnrolledInMDM() (bool, string, error) {
	return false, "", ErrNotImplemented
}

func CheckAssignedEnrollmentProfile(expectedURL string) error {
	return ErrNotImplemented
}

func GetCustomEnrollmentProfileEndUserEmail() (string, error) {
	return "", ErrNotImplemented
}
