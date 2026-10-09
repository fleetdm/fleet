//go:build darwin || linux

package sentinelone

import "os"

// firstExistingPath returns the first path that exists, or "" if none do.
func firstExistingPath(paths []string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
