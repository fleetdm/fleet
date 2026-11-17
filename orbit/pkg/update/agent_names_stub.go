//go:build !linux && !windows

package update

// applyAgentDisplayNames does nothing on macOS: Activity Monitor names a process after its executable file, and
// renaming it would break Full Disk Access profiles that match orbit by path.
func applyAgentDisplayNames() error {
	return nil
}
