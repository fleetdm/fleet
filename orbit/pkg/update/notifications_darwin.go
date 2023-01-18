//go:build darwin

package update

func runRenewEnrollmentProfile() error {
	cmd := exec.Command("/usr/bin/profiles", "renew", "--type", "enrollment")
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) > 0 {
		// just as a precaution, limit the length of the output
		if len(out) > 512 {
			out = out[:512]
		}
		err = fmt.Errorf("%w: %s", err, string(out))
	}
	return err
}
