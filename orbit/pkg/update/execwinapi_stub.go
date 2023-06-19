//go:build !windows

package update

func RunWindowsMDMEnrollment() error {
	return nil
}
