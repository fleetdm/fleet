//go:build !darwin

package main

// resetControlCenter is a no-op on non-Darwin platforms.
func resetControlCenter() {}

// promptMenuBarAccess is a no-op on non-Darwin platforms.
func promptMenuBarAccess() {}
