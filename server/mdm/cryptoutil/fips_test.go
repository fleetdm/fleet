package cryptoutil

import (
	"os"
	"os/exec"
	"testing"
)

func TestIsFIPSModeEnabled(t *testing.T) {
	// This test runs as a subprocess with GODEBUG set to test the enabled path.
	// We need a subprocess because sync.Once caches the result.
	if os.Getenv("TEST_FIPS_SUBPROCESS") == "1" {
		if !IsFIPSMode() {
			os.Exit(1)
		}
		os.Exit(0)
	}

	t.Run("disabled", func(t *testing.T) {
		// In normal test run, GODEBUG is not set, so FIPS mode is disabled.
		if IsFIPSMode() {
			t.Error("expected FIPS mode to be disabled")
		}
	})

	t.Run("enabled", func(t *testing.T) {
		// Run subprocess with GODEBUG=fips140=only
		cmd := exec.Command(os.Args[0], "-test.run=TestIsFIPSModeEnabled")
		cmd.Env = append(os.Environ(), "GODEBUG=fips140=only", "TEST_FIPS_SUBPROCESS=1")
		if err := cmd.Run(); err != nil {
			t.Error("expected FIPS mode to be enabled with GODEBUG=fips140=only")
		}
	})
}
