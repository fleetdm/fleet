//go:build darwin

package update

func runRenewEnrollmentProfile() error {
	return runCmdCollectErr("/usr/bin/profiles", "renew", "--type", "enrollment")
}

func runKickstartSoftwareUpdated() error {
	return runCmdCollectErr("launchctl", "kickstart", "-k", "system/com.apple.softwareupdated")
}
