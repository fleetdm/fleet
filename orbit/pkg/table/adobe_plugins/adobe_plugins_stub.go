//go:build !darwin && !windows

package adobe_plugins

import "github.com/rs/zerolog"

// getScanPaths returns nil on unsupported platforms (Linux, etc.).
// Adobe Creative Cloud does not run on Linux.
func getScanPaths(_ string, _ zerolog.Logger) ([]scanPath, error) {
	return nil, nil
}
