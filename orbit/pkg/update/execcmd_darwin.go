//go:build darwin

package update

func runRenewEnrollmentProfile() error {
	return runCmdCollectErr("/usr/bin/profiles", "renew", "--type", "enrollment")
}
