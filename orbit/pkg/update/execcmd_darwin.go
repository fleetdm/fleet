//go:build darwin

package update

func runRenewEnrollmentProfile() error {
	cmd := `launchctl asuser $(id -u $(stat -f "%u" /dev/console)) profiles renew -type enrollment`
	return runCmdCollectErr("sh", "-c", cmd)
}
