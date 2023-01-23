//go:build !windows
// +build !windows

package table

// zerotierCli returns the path to the CLI executable.
func zerotierCli(args ...string) []string {
	cmd := []string{"/usr/local/bin/zerotier-cli"}

	return append(cmd, args...)
}
