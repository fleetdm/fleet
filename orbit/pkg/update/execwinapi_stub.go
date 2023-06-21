//go:build !windows

package update

func RunMicrosoftMDMEnrollment(args MicrosoftMDMEnrollmentArgs) error {
	return nil
}

func RunWindowsMDMUnenrollment(args WindowsMDMEnrollmentArgs) error {
	return nil
}
