//go:build !darwin

package update

// UsesFleetPath is only true on macOS, where the executable's path carries the agent's name.
func UsesFleetPath(string) bool { return false }

// ReexecAsFleet does nothing outside macOS.
func ReexecAsFleet(string) {}

func ensureAgentPath(string) (bool, error) { return false, nil }
