//go:build !windows

package update

func updateUninstallFleetdRegistryVersion(newVersion string) error {
	return nil
}
