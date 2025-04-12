//go:build !windows

package update

func updateRegistryVersion(newVersion string) error {
	return nil
}
