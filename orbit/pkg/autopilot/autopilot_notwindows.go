//go:build !windows

package autopilot

// RegistrationID returns an empty string, since only Windows devices are registered in Autopilot.
func RegistrationID() (string, error) {
	return "", nil
}
