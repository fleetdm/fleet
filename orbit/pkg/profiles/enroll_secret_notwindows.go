//go:build !windows

package profiles

// GetEnrollSecret is not implemented outside Windows. macOS carries its enroll secret in the fleetd
// configuration profile instead, read via GetFleetdConfig.
func GetEnrollSecret() (string, error) {
	return "", ErrNotImplemented
}

// ClearEnrollSecret is not implemented outside Windows.
func ClearEnrollSecret() error {
	return ErrNotImplemented
}
