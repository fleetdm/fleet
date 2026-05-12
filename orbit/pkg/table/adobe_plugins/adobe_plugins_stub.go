//go:build !darwin && !windows

package adobe_plugins

// getScanPaths returns nil on unsupported platforms (Linux, etc.).
// Adobe Creative Cloud does not run on Linux.
func getScanPaths(_ string) ([]scanPath, error) {
	return nil, nil
}
