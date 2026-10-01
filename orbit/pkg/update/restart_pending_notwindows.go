//go:build !windows

package update

func isRestartPending() (bool, error) { return false, nil }
