//go:build !darwin

package endpoint_verification_accounts

import (
	"fmt"
	"runtime"
)

// listLocalUsersDarwin is a stub on non-darwin platforms. The endpoint
// verification accounts table is macOS-only in v1; Windows and Linux
// support is pending verification of EV's on-disk layout on those
// platforms (Google's REST docs name `~/.secureConnect/context_aware_config.json`
// but the macOS path has already been observed to be stale, so the other
// platforms warrant confirmation against a live EV install before shipping).
//
// This stub exists so the package compiles on non-darwin targets — orbit
// is built for darwin, linux, and windows even though the resolution
// table only emits rows on darwin in this release.
func listLocalUsersDarwin() ([]userHome, error) {
	return nil, fmt.Errorf("endpoint_verification_accounts: %s not yet implemented", runtime.GOOS)
}
