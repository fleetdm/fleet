package config

import (
	"os"
	"strings"
	"testing"
)

// RestoreEnv restores the environment variables from the saved slice.
// Do _not_ use this in parallel tests, as it clears the entire environment.
func RestoreEnv(t *testing.T, saved []string) {
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
}
