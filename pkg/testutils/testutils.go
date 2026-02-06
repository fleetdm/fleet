package testutils

import (
	"os"
	"strings"
	"testing"
)

// TestLogWriter adapts testing.TB to io.Writer for use with go-kit/log.
// Logs are associated with the test and only shown on failure (or with -v).
type TestLogWriter struct {
	T testing.TB
}

func (w *TestLogWriter) Write(p []byte) (n int, err error) {
	// Trim trailing newline because go-kit/log adds one, and t.Log() adds another.
	w.T.Log(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}

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
