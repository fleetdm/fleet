package service

import "testing"

// TestCICanaryFail is a temporary test to verify CI failure reporting.
// Remove after confirming CI behavior.
func TestCICanaryFail(t *testing.T) {
	t.Fatal("intentional failure: CI canary test for service suite")
}
