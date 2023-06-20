//go:build !windows

package update

func RunWindowsMDMEnrollment(discoveryURL string) error {
	return nil
}
