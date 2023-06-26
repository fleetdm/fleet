//go:build !windows

package update

func RunMicrosoftMDMEnrollment(args MicrosoftMDMEnrollmentArgs) error {
	return nil
}

func RunMicrosoftMDMUnenrollment(args MicrosoftMDMEnrollmentArgs) error {
	return nil
}
