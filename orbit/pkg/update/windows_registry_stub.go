//go:build !windows

package update

func updateUninstallFleetdRegistryVersion(newVersion string) error {
	return nil
}

func restoreUninstallFleetdRegistryName() error {
	return nil
}
