package testutils

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"unicode/utf16"
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

// UTF16FromString returns the UTF-16 encoding of the UTF-8 string s, with a terminating NUL added.
// If s contains a NUL byte at any location, it returns (nil, syscall.EINVAL).
func UTF16FromString(s string) ([]uint16, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return nil, syscall.EINVAL
		}
	}
	return utf16.Encode([]rune(s + "\x00")), nil
}
