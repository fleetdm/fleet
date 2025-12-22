package testutils

import (
	"os"
	"strings"
	"testing"
)

// SaveEnv snapshots the current environment and restores it when the test
// ends.
//
// Do _not_ use this in parallel tests, as it clears the entire environment.
func SaveEnv(t *testing.T) {
	saved := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, kv := range saved {
			parts := strings.SplitN(kv, "=", 2)
			key := parts[0]
			val := ""
			if len(parts) == 2 {
				val = parts[1]
			}
			err := os.Setenv(key, val)
			if err != nil {
				t.Logf("error restoring env var %s: %v", key, err)
			}
		}
	})
}
