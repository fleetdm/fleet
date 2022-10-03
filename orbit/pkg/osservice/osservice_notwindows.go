//go:build !windows
// +build !windows

package osservice

// SetupServiceManagement is currently a placeholder for non-windows OSes
// system service configuration
func SetupServiceManagement(serviceName string, interruptCh chan struct{}, doneCh chan struct{}) {
}
