//go:build windows

package update

import "golang.org/x/sys/windows/registry"

// isRestartPending reports whether Windows has a restart staged.
func isRestartPending() (bool, error) {
	for _, key := range []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`,
	} {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.QUERY_VALUE)
		if err == nil {
			k.Close()
			return true, nil
		}
		if err != registry.ErrNotExist {
			return false, err
		}
	}
	return false, nil
}
