//go:build !darwin

package processes

// waitExit can't watch an exit without reaping off macOS, Hangar's only
// platform, so callers take their paths that never signal a reaped group.
func waitExit(int) bool { return false }
