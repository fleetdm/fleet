//go:build !linux && !windows

package update

// applyAgentDisplayNames does nothing on macOS, where the process name comes from the executable's path; see
// AgentPathReceiver.
func applyAgentDisplayNames() error {
	return nil
}
