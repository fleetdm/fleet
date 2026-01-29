package cryptoutil

import (
	"os"
	"strings"
	"sync"
)

var (
	fipsMode bool
	fipsOnce sync.Once
)

// IsFIPSMode returns true if FIPS 140-3 mode is enabled via GODEBUG.
// It checks for GODEBUG=fips140=only or GODEBUG=fips140=on.
func IsFIPSMode() bool {
	fipsOnce.Do(func() {
		godebug := os.Getenv("GODEBUG")
		fipsMode = strings.Contains(godebug, "fips140=only") ||
			strings.Contains(godebug, "fips140=on")
	})
	return fipsMode
}
