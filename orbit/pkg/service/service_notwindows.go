//go:build !windows
// +build !windows

package service

// This is currently a placeholder for non-windows OSes
// system service configuration
func SetupServiceManagement(serviceName string, serviceRootDir string) error {
	return nil
}
